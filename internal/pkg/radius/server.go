package radius

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2759"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc2869"
	"layeh.com/radius/vendors/microsoft"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

type Server struct {
	customerRepo repository.CustomerRepository
	sessionRepo  repository.SessionRepository
	routerRepo   repository.RouterRepository
	voucherRepo  repository.VoucherRepository
	ipamRepo     repository.IPAMRepository
	secret       string
}

func NewServer(
	customerRepo repository.CustomerRepository,
	sessionRepo repository.SessionRepository,
	routerRepo repository.RouterRepository,
	voucherRepo repository.VoucherRepository,
	ipamRepo repository.IPAMRepository,
	secret string,
) *Server {
	return &Server{
		customerRepo: customerRepo,
		sessionRepo:  sessionRepo,
		routerRepo:   routerRepo,
		voucherRepo:  voucherRepo,
		ipamRepo:     ipamRepo,
		secret:       secret,
	}
}

// perRouterSecretSource looks up the RADIUS secret per router (by NAS-IP-Address).
// Falls back to the global secret if router not found.
type perRouterSecretSource struct {
	routerRepo    repository.RouterRepository
	defaultSecret []byte
}

func (s *perRouterSecretSource) RADIUSSecret(ctx context.Context, remoteAddr net.Addr) ([]byte, error) {
	host, _, _ := net.SplitHostPort(remoteAddr.String())
	router, err := s.routerRepo.FindByVPNIP(ctx, host)
	if err == nil && router != nil && router.RADIUSSecret != "" {
		return []byte(router.RADIUSSecret), nil
	}
	return s.defaultSecret, nil
}

func (s *Server) StartAuth(addr string) error {
	server := radius.PacketServer{
		Addr:    addr,
		Network: "udp",
		SecretSource: &perRouterSecretSource{
			routerRepo:    s.routerRepo,
			defaultSecret: []byte(s.secret),
		},
		Handler: radius.HandlerFunc(s.handleAuth),
	}
	log.Printf("RADIUS Auth server listening on %s", addr)
	return server.ListenAndServe()
}

func (s *Server) StartAcct(addr string) error {
	server := radius.PacketServer{
		Addr:    addr,
		Network: "udp",
		SecretSource: &perRouterSecretSource{
			routerRepo:    s.routerRepo,
			defaultSecret: []byte(s.secret),
		},
		Handler: radius.HandlerFunc(s.handleAccounting),
	}
	log.Printf("RADIUS Acct server listening on %s", addr)
	return server.ListenAndServe()
}

// authMethod indicates which authentication method was used.
type authMethod int

const (
	authNone    authMethod = 0
	authPAP     authMethod = 1
	authCHAP    authMethod = 2
	authMSCHAP2 authMethod = 3
	authMSCHAP1 authMethod = 4
)

// mschapv2Result holds MS-CHAPv2 auth data needed for the Access-Accept response.
type mschapv2Result struct {
	success []byte
}

// detectAuthMethod returns which auth method the request uses.
func detectAuthMethod(r *radius.Request) authMethod {
	if rfc2865.UserPassword_GetString(r.Packet) != "" {
		return authPAP
	}
	if len(rfc2865.CHAPPassword_Get(r.Packet)) == 17 {
		return authCHAP
	}
	challenge := microsoft.MSCHAPChallenge_Get(r.Packet)
	response := microsoft.MSCHAP2Response_Get(r.Packet)
	if len(challenge) == 16 && len(response) == 50 {
		return authMSCHAP2
	}
	v1response := microsoft.MSCHAPResponse_Get(r.Packet)
	if len(challenge) == 8 && len(v1response) == 50 {
		return authMSCHAP1
	}
	return authNone
}

// verifyPAP checks PAP authentication.
func verifyPAP(r *radius.Request, storedPassword string) bool {
	return rfc2865.UserPassword_GetString(r.Packet) == storedPassword
}

// verifyCHAP checks standard CHAP authentication.
func verifyCHAP(r *radius.Request, storedPassword string) bool {
	chapPassword := rfc2865.CHAPPassword_Get(r.Packet)
	chapID := chapPassword[0]
	chapHash := chapPassword[1:]

	challenge := rfc2865.CHAPChallenge_Get(r.Packet)
	if len(challenge) == 0 {
		challenge = r.Packet.Authenticator[:]
	}

	h := md5.New()
	h.Write([]byte{chapID})
	h.Write([]byte(storedPassword))
	h.Write(challenge)
	return bytes.Equal(chapHash, h.Sum(nil))
}

// verifyMSCHAPv2 checks MS-CHAPv2 authentication and returns success data for Access-Accept.
func verifyMSCHAPv2(r *radius.Request, username, storedPassword string) (bool, *mschapv2Result) {
	challenge := microsoft.MSCHAPChallenge_Get(r.Packet)
	response := microsoft.MSCHAP2Response_Get(r.Packet)

	ident := response[0]
	peerChallenge := response[2:18]
	peerResponse := response[26:50]

	user := []byte(username)
	pass := []byte(storedPassword)

	ntResponse, err := rfc2759.GenerateNTResponse(challenge, peerChallenge, user, pass)
	if err != nil {
		return false, nil
	}

	if !bytes.Equal(ntResponse, peerResponse) {
		return false, nil
	}

	authResp, err := rfc2759.GenerateAuthenticatorResponse(challenge, peerChallenge, ntResponse, user, pass)
	if err != nil {
		return false, nil
	}

	success := make([]byte, 43)
	success[0] = ident
	copy(success[1:], authResp)

	return true, &mschapv2Result{success: success}
}

// verifyMSCHAPv1 checks MS-CHAPv1 authentication using the NT-Challenge-Response.
func verifyMSCHAPv1(r *radius.Request, storedPassword string) bool {
	challenge := microsoft.MSCHAPChallenge_Get(r.Packet)
	response := microsoft.MSCHAPResponse_Get(r.Packet)

	// MS-CHAPv1 response layout (50 bytes):
	//   [0]     = ident
	//   [1]     = flags (1 = use NT response)
	//   [2:26]  = LM-Challenge-Response (24 bytes)
	//   [26:50] = NT-Challenge-Response (24 bytes)
	ntResponse := response[26:50]

	// Convert password to UTF-16LE before hashing (same as MS-CHAPv2 internals)
	ucs2Password, err := rfc2759.ToUTF16([]byte(storedPassword))
	if err != nil {
		return false
	}
	passwordHash := rfc2759.NTPasswordHash(ucs2Password)
	expected := rfc2759.ChallengeResponse(challenge, passwordHash)

	return bytes.Equal(ntResponse, expected)
}

func (s *Server) handleAuth(w radius.ResponseWriter, r *radius.Request) {
	username := rfc2865.UserName_GetString(r.Packet)
	nasIP := rfc2865.NASIPAddress_Get(r.Packet)
	method := detectAuthMethod(r)

	// AUTH REQUEST log removed — too noisy (called on every PPPoE connect attempt)

	ctx := r.Context()

	// Only accept auth from registered routers
	router, _ := s.routerRepo.FindByVPNIP(ctx, nasIP.String())
	if router == nil {
		log.Printf("RADIUS AUTH REJECT: unknown NAS %s user=%s", nasIP, username)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	if method == authNone {
		log.Printf("RADIUS AUTH REJECT: user=%s unsupported_auth_method nas=%s", username, nasIP)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	// Try PPPoE customer first
	customer, err := s.customerRepo.FindByPPPoEUsername(ctx, router.TenantID, username)
	if err == nil && customer != nil {
		s.handleCustomerAuth(w, r, customer, nasIP, method, router.RouterType)
		return
	}

	// Try voucher hotspot user
	voucher, err := s.voucherRepo.FindByUsernameWithProduct(ctx, username)
	if err == nil && voucher != nil {
		s.handleVoucherAuth(w, r, voucher, nasIP, method, router.RouterType)
		return
	}

	log.Printf("RADIUS AUTH REJECT: user=%s not_found nas=%s", username, nasIP)
	_ = w.Write(r.Response(radius.CodeAccessReject))
}

// applyRateLimit adds the bandwidth/speed-limit attributes appropriate for the
// router brand. Authentication, IP assignment, accounting and CoA are standard
// RADIUS and identical for every vendor — only the rate-limit attribute differs.
//
//	mikrotik / vyos        : Mikrotik-Rate-Limit VSA (accel-ppp on VyOS/EdgeRouter reads it too)
//	huawei                 : Huawei-Input/Output-Average-Rate VSAs (bit/s)
//	cisco / ruijie / juniper : Filter-Id referencing a NAS-local QoS policy named after the package
//	                           (Cisco/Ruijie also receive ip:sub-qos-policy AVPairs)
//
// Returns a short human-readable description for logging.
func (s *Server) applyRateLimit(resp *radius.Packet, routerType string, upMbps, downMbps int, addressList, packageName string) string {
	switch routerType {
	case "huawei":
		resp.Add(26, radius.Attribute(EncodeVSAInt(HuaweiVendorID, HuaweiAttrInputAverageRate, uint32(upMbps)*1_000_000)))
		resp.Add(26, radius.Attribute(EncodeVSAInt(HuaweiVendorID, HuaweiAttrOutputAverageRate, uint32(downMbps)*1_000_000)))
		return fmt.Sprintf("huawei %dM/%dM", upMbps, downMbps)

	case "cisco", "ruijie", "juniper":
		policy := sanitizePolicyName(packageName)
		if policy != "" {
			resp.Add(11, radius.Attribute([]byte(policy))) // Filter-Id (RFC 2865)
		}
		if routerType != "juniper" && policy != "" {
			resp.Add(26, radius.Attribute(EncodeVSA(CiscoVendorID, CiscoAttrAVPair, "ip:sub-qos-policy-in="+policy)))
			resp.Add(26, radius.Attribute(EncodeVSA(CiscoVendorID, CiscoAttrAVPair, "ip:sub-qos-policy-out="+policy)))
		}
		return fmt.Sprintf("%s filter-id=%s", routerType, policy)

	default: // mikrotik, vyos, or unknown → MikroTik-style inline rate
		rate := fmt.Sprintf("%dM/%dM", upMbps, downMbps)
		if upMbps > 0 && downMbps > 0 {
			rate = fmt.Sprintf("%dM/%dM %dM/%dM %dM/%dM 30/30",
				upMbps, downMbps,
				model.AutoBurstMbps(upMbps), model.AutoBurstMbps(downMbps),
				model.BurstThresholdMbps(upMbps), model.BurstThresholdMbps(downMbps))
		}
		resp.Add(26, radius.Attribute(EncodeVSA(MikroTikVendorID, MikroTikAttrRateLimit, rate)))
		if addressList != "" {
			resp.Add(26, radius.Attribute(EncodeVSA(MikroTikVendorID, MikroTikAttrAddressList, addressList)))
		}
		return rate
	}
}

// sanitizePolicyName makes a package name safe to use as a NAS policy/filter name.
func sanitizePolicyName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('_')
		}
	}
	return b.String()
}

func (s *Server) handleCustomerAuth(w radius.ResponseWriter, r *radius.Request, customer *model.Customer, nasIP net.IP, method authMethod, routerType string) {
	username := customer.PPPoEUsername

	// Validate password based on auth method
	var authOK bool
	var mschap2 *mschapv2Result

	switch method {
	case authPAP:
		authOK = verifyPAP(r, customer.PPPoEPassword)
	case authCHAP:
		authOK = verifyCHAP(r, customer.PPPoEPassword)
	case authMSCHAP2:
		authOK, mschap2 = verifyMSCHAPv2(r, username, customer.PPPoEPassword)
	case authMSCHAP1:
		authOK = verifyMSCHAPv1(r, customer.PPPoEPassword)
	}

	if !authOK {
		log.Printf("RADIUS AUTH REJECT: user=%s wrong_password nas=%s", username, nasIP)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	// Check status
	if customer.Status != "active" {
		log.Printf("RADIUS AUTH REJECT: user=%s status=%s", username, customer.Status)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	// Check package exists and is active
	if customer.Package == nil {
		log.Printf("RADIUS AUTH REJECT: user=%s no_package", username)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}
	if !customer.Package.IsActive {
		log.Printf("RADIUS AUTH REJECT: user=%s package=%s package_inactive", username, customer.Package.Name)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	// Build Access-Accept response
	resp := r.Response(radius.CodeAccessAccept)

	// Assign Framed-IP-Address from customer IP or IPAM pool
	framedIP := customer.IPAddress
	if framedIP == "" && customer.RouterID != nil {
		pool, _ := s.ipamRepo.FindPoolByRouterID(r.Context(), *customer.RouterID)
		if pool != nil {
			// Check if customer already has an IP in this exact pool — reuse it and
			// clean up any stale IPs from other pools in one shot.
			existing, _ := s.ipamRepo.FindByCustomerAndPool(r.Context(), customer.ID, pool.ID)
			if existing != nil {
				// Reuse existing IP. Clean up any stale IPs in other pools.
				_ = s.ipamRepo.ReleaseAllByCustomerIDExcept(r.Context(), customer.ID, existing.ID)
				framedIP = existing.Address
				log.Printf("RADIUS IPAM: reusing assigned IP %s for user=%s", framedIP, username)
			} else {
				// No valid IPAM assignment for this pool — allocate new IP atomically.
				// ReleaseAndAssign releases ALL previous IPs (across all pools) then assigns fresh.
				addr, _ := s.ipamRepo.ReleaseAndAssign(r.Context(), pool.TenantID, pool.ID, customer.ID)
				if addr != nil {
					framedIP = addr.Address
					log.Printf("RADIUS IPAM: auto-assigned %s to user=%s from pool=%s", framedIP, username, pool.Name)
				}
			}
		}
	}
	if framedIP != "" {
		ip := net.ParseIP(framedIP)
		if ip != nil {
			rfc2865.FramedIPAddress_Add(resp, ip)
		}
	}

	// Apply per-vendor bandwidth/speed limit (vendor-specific attributes).
	rateLimit := s.applyRateLimit(resp, routerType,
		customer.Package.BandwidthUp, customer.Package.BandwidthDown,
		customer.Package.AddressList, customer.Package.Name)

	// Add MS-CHAP2-Success attribute for MS-CHAPv2 auth
	if method == authMSCHAP2 && mschap2 != nil {
		microsoft.MSCHAP2Success_Add(resp, mschap2.success)
	}

	methodName := map[authMethod]string{authPAP: "pap", authCHAP: "chap", authMSCHAP2: "mschapv2"}[method]
	log.Printf("RADIUS AUTH ACCEPT: user=%s package=%s rate=[%s] nas=%s type=%s (%s)",
		username, customer.Package.Name, rateLimit, nasIP, routerType, methodName)

	_ = w.Write(resp)
}

func (s *Server) handleVoucherAuth(w radius.ResponseWriter, r *radius.Request, voucher *model.Voucher, nasIP net.IP, method authMethod, routerType string) {
	username := voucher.Username

	// Validate password based on auth method
	var authOK bool
	var mschap2 *mschapv2Result

	switch method {
	case authPAP:
		authOK = verifyPAP(r, voucher.Password)
	case authCHAP:
		authOK = verifyCHAP(r, voucher.Password)
	case authMSCHAP2:
		authOK, mschap2 = verifyMSCHAPv2(r, username, voucher.Password)
	case authMSCHAP1:
		authOK = verifyMSCHAPv1(r, voucher.Password)
	}

	if !authOK {
		log.Printf("RADIUS AUTH REJECT: voucher=%s wrong_password nas=%s", username, nasIP)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	// Check voucher status — must be "sold" or "active"
	if voucher.Status != "sold" && voucher.Status != "active" {
		log.Printf("RADIUS AUTH REJECT: voucher=%s status=%s", username, voucher.Status)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	// Check expiry for active vouchers
	if voucher.Status == "active" && voucher.ExpiresAt != nil {
		if time.Now().After(*voucher.ExpiresAt) {
			log.Printf("RADIUS AUTH REJECT: voucher=%s expired", username)
			_ = s.voucherRepo.UpdateStatus(r.Context(), voucher.ID, "expired")
			_ = w.Write(r.Response(radius.CodeAccessReject))
			return
		}
	}

	// Auto-activate on first login if sold
	if voucher.Status == "sold" && voucher.Product != nil {
		now := time.Now()
		expiresAt := now.Add(time.Duration(voucher.Product.Duration) * 24 * time.Hour)
		_ = s.voucherRepo.Activate(r.Context(), voucher.ID, now, expiresAt)
		// Reflect expiry in-memory so the Session-Timeout below is set on THIS
		// first session (Activate only persists it to the DB).
		voucher.ExpiresAt = &expiresAt
		log.Printf("RADIUS voucher auto-activated: voucher=%s expires=%s", username, expiresAt.Format(time.RFC3339))
	}

	if voucher.Product == nil {
		log.Printf("RADIUS AUTH REJECT: voucher=%s no_product", username)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	// Build Access-Accept response
	resp := r.Response(radius.CodeAccessAccept)

	// Apply per-vendor bandwidth/speed limit.
	rateLimit := s.applyRateLimit(resp, routerType,
		voucher.Product.BandwidthUp, voucher.Product.BandwidthDown, "", voucher.Product.Name)

	// Set session timeout: remaining time until expiry
	if voucher.ExpiresAt != nil {
		remaining := time.Until(*voucher.ExpiresAt)
		if remaining > 0 {
			_ = rfc2865.SessionTimeout_Set(resp, rfc2865.SessionTimeout(uint32(remaining.Seconds())))
		}
	}

	// Add MS-CHAP2-Success attribute for MS-CHAPv2 auth
	if method == authMSCHAP2 && mschap2 != nil {
		microsoft.MSCHAP2Success_Add(resp, mschap2.success)
	}

	log.Printf("RADIUS AUTH ACCEPT: voucher=%s product=%s rate=[%s] nas=%s type=%s",
		username, voucher.Product.Name, rateLimit, nasIP, routerType)

	_ = w.Write(resp)
}

// handleAccounting processes Accounting-Request packets from MikroTik NAS.
func (s *Server) handleAccounting(w radius.ResponseWriter, r *radius.Request) {
	username := rfc2865.UserName_GetString(r.Packet)
	acctType := rfc2866.AcctStatusType_Get(r.Packet)
	sessionID := rfc2866.AcctSessionID_GetString(r.Packet)
	nasIP := rfc2865.NASIPAddress_Get(r.Packet)
	framedIP := rfc2865.FramedIPAddress_Get(r.Packet)
	callerID := rfc2865.CallingStationID_GetString(r.Packet)
	inputOctets := rfc2866.AcctInputOctets_Get(r.Packet)
	outputOctets := rfc2866.AcctOutputOctets_Get(r.Packet)
	sessionTime := rfc2866.AcctSessionTime_Get(r.Packet)

	ctx := r.Context()

	// Only accept accounting from registered routers
	router, _ := s.routerRepo.FindByVPNIP(ctx, nasIP.String())
	if router == nil {
		log.Printf("RADIUS ACCT REJECT: unknown NAS %s user=%s session=%s", nasIP, username, sessionID)
		_ = w.Write(r.Response(radius.CodeAccountingResponse))
		return
	}

	// Always sync nas_ip from accounting packet — WAN IP changes on PPPoE reconnect
	// so this is more up-to-date than waiting for the next heartbeat interval.
	// Tunnel IPs (WireGuard / legacy L2TP-SSTP) must never overwrite nas_ip:
	// nas_ip is the router's WAN address, used for CoA in Direct mode only.
	if ip := nasIP.String(); ip != "" && ip != router.VPNIP && ip != router.LegacyVPNIP {
		_ = s.routerRepo.UpdateNASIP(ctx, router.ID, ip)
	}

	// Octet counters are 32-bit in RADIUS; the high 32 bits are carried in the
	// Gigawords attributes. Combine them so sessions >4 GB aren't truncated.
	totalIn := int64(rfc2869.AcctInputGigawords_Get(r.Packet))<<32 + int64(inputOctets)
	totalOut := int64(rfc2869.AcctOutputGigawords_Get(r.Packet))<<32 + int64(outputOctets)

	switch acctType {
	case rfc2866.AcctStatusType_Value_Start:
		s.handleAcctStart(ctx, username, sessionID, nasIP.String(), framedIP.String(), callerID)

	case rfc2866.AcctStatusType_Value_InterimUpdate:
		s.handleAcctInterim(ctx, username, sessionID, nasIP.String(), framedIP.String(), callerID, totalIn, totalOut, uint32(sessionTime))

	case rfc2866.AcctStatusType_Value_Stop:
		terminateCause := rfc2866.AcctTerminateCause_Get(r.Packet)
		s.handleAcctStop(ctx, username, sessionID, terminateCause.String(), totalIn, totalOut, uint32(sessionTime))
	}

	// Always respond with Accounting-Response
	_ = w.Write(r.Response(radius.CodeAccountingResponse))
}

func (s *Server) handleAcctStart(ctx context.Context, username, sessionID, nasIP, framedIP, callerID string) {
	// Find customer and router for tenant context
	router, _ := s.routerRepo.FindByVPNIP(ctx, nasIP)
	tenantID := ""
	if router != nil {
		tenantID = router.TenantID
	}
	customer, _ := s.customerRepo.FindByPPPoEUsername(ctx, tenantID, username)

	// Only accept sessions from users registered in this system (customer or voucher)
	var voucherUser bool
	if customer == nil {
		voucher, _ := s.voucherRepo.FindByUsernameWithProduct(ctx, username)
		voucherUser = voucher != nil
	}
	if customer == nil && !voucherUser {
		log.Printf("RADIUS ACCT SKIP: user=%s not registered in this system nas=%s session=%s", username, nasIP, sessionID)
		return
	}

	// Close any previous active session for this user to prevent ghost sessions and IP leaks
	oldSession, _ := s.sessionRepo.FindActiveByUsername(ctx, tenantID, username)
	if oldSession != nil && oldSession.SessionID != sessionID {
		_ = s.sessionRepo.EndSession(ctx, oldSession.SessionID, "NAS-Reset", 0, 0, 0)
		log.Printf("RADIUS ACCT START: closed stale session=%s for user=%s (replaced by %s)", oldSession.SessionID, username, sessionID)

		// Send CoA Disconnect-Request to MikroTik to force-remove the old PPPoE interface
		// This prevents ghost interfaces (e.g. <pppoe-username-1>) on the router
		if router != nil && router.CoAPort > 0 && router.RADIUSSecret != "" {
			go func(nas string, port int, secret, user, sess string) {
				if err := DisconnectUser(nas, port, secret, user, sess); err != nil {
					log.Printf("RADIUS ACCT START: failed to disconnect old session=%s on NAS: %v", sess, err)
				}
			}(oldSession.NASIPAddress, router.CoAPort, router.RADIUSSecret, username, oldSession.SessionID)
		}
	}

	session := &model.RadiusSession{
		SessionID:    sessionID,
		Username:     username,
		NASIPAddress: nasIP,
		FramedIP:     framedIP,
		CallerID:     callerID,
		StartedAt:    time.Now(),
		Status:       "active",
	}

	if customer != nil {
		session.TenantID = customer.TenantID
		session.CustomerID = &customer.ID
	}
	if router != nil {
		session.RouterID = &router.ID
		if session.TenantID == "" {
			session.TenantID = router.TenantID
		}
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		log.Printf("RADIUS ACCT START ERROR: session=%s user=%s err=%v", sessionID, username, err)
		return
	}

	log.Printf("RADIUS ACCT START: session=%s user=%s nas=%s ip=%s", sessionID, username, nasIP, framedIP)
}

func (s *Server) handleAcctInterim(ctx context.Context, username, sessionID, nasIP, framedIP, callerID string, inputOctets, outputOctets int64, sessionTime uint32) {
	prev, err := s.sessionRepo.UpdateUsage(ctx, sessionID, inputOctets, outputOctets, sessionTime)
	if err != nil {
		// Session not found as active — re-create it (e.g. after VPS reboot)
		log.Printf("RADIUS ACCT INTERIM RECOVER: session=%s user=%s — re-creating session", sessionID, username)
		s.handleAcctStart(ctx, username, sessionID, nasIP, framedIP, callerID)
		// Retry the update after re-creation
		prev, err = s.sessionRepo.UpdateUsage(ctx, sessionID, inputOctets, outputOctets, sessionTime)
		if err != nil {
			log.Printf("RADIUS ACCT INTERIM ERROR: session=%s err=%v", sessionID, err)
			return
		}
	}

	// Compute delta bytes and interval for bandwidth sample
	deltaIn := inputOctets - prev.PrevInput
	deltaOut := outputOctets - prev.PrevOutput
	deltaSec := int(sessionTime) - prev.PrevSessionTime

	if deltaIn < 0 {
		deltaIn = inputOctets
	}
	if deltaOut < 0 {
		deltaOut = outputOctets
	}

	if deltaSec > 0 {
		uploadBps := deltaIn * 8 / int64(deltaSec)
		downloadBps := deltaOut * 8 / int64(deltaSec)
		if err := s.sessionRepo.InsertBandwidthSample(ctx, prev.TenantID, prev.CustomerID, sessionID, deltaSec, downloadBps, uploadBps); err != nil {
			log.Printf("RADIUS BW SAMPLE ERROR: session=%s err=%v", sessionID, err)
		}
	}

	// INTERIM log removed — too noisy (called every ~60s per active session)
}

func (s *Server) handleAcctStop(ctx context.Context, username, sessionID, terminateCause string, inputOctets, outputOctets int64, sessionTime uint32) {
	if err := s.sessionRepo.EndSession(ctx, sessionID, terminateCause, inputOctets, outputOctets, sessionTime); err != nil {
		log.Printf("RADIUS ACCT STOP ERROR: session=%s err=%v", sessionID, err)
		return
	}

	// Do NOT release IPAM IP here — let ReleaseAndAssign handle it during next auth.
	// This avoids a race condition where acct-stop arrives between auth and acct-start
	// of a reconnecting session, which would incorrectly release the IP.

	log.Printf("RADIUS ACCT STOP: session=%s cause=%s in=%d out=%d time=%ds",
		sessionID, terminateCause, inputOctets, outputOctets, sessionTime)
}

// DisconnectUser sends a Disconnect-Request (CoA) to MikroTik NAS
// to terminate an active PPPoE session. Waits for Disconnect-ACK/NAK response.
func DisconnectUser(nasAddr string, coaPort int, secret, username, sessionID string) error {
	packet := radius.New(radius.CodeDisconnectRequest, []byte(secret))
	_ = rfc2865.UserName_SetString(packet, username)
	_ = rfc2866.AcctSessionID_SetString(packet, sessionID)

	addr := net.JoinHostPort(nasAddr, fmt.Sprintf("%d", coaPort))
	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer conn.Close()

	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	encoded, err := packet.Encode()
	if err != nil {
		return fmt.Errorf("encode packet: %w", err)
	}

	_, err = conn.Write(encoded)
	if err != nil {
		return fmt.Errorf("write to %s: %w", addr, err)
	}

	// Read Disconnect-ACK/NAK response
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		// Timeout or read error — log but don't fail (NAS may not respond)
		log.Printf("RADIUS CoA DISCONNECT sent (no response): user=%s session=%s nas=%s err=%v", username, sessionID, addr, err)
		return nil
	}

	resp, err := radius.Parse(buf[:n], []byte(secret))
	if err != nil {
		log.Printf("RADIUS CoA DISCONNECT sent (invalid response): user=%s session=%s nas=%s", username, sessionID, addr)
		return nil
	}

	if resp.Code == radius.CodeDisconnectACK {
		log.Printf("RADIUS CoA DISCONNECT-ACK: user=%s session=%s nas=%s", username, sessionID, addr)
	} else {
		log.Printf("RADIUS CoA DISCONNECT-NAK: user=%s session=%s nas=%s code=%d", username, sessionID, addr, resp.Code)
	}

	return nil
}
