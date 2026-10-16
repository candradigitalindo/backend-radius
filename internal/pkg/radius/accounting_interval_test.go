package radius

import (
	"net"
	"testing"

	"github.com/candrasyahputra/radius-server/internal/model"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
)

type capturedResponse struct{ packet *radius.Packet }

func (w *capturedResponse) Write(packet *radius.Packet) error {
	w.packet = packet
	return nil
}

func TestAcceptedLoginsRequestFiveMinuteAccounting(t *testing.T) {
	for _, kind := range []string{"customer", "voucher"} {
		t.Run(kind, func(t *testing.T) {
			request := &radius.Request{Packet: radius.New(radius.CodeAccessRequest, []byte("shared-secret"))}
			if err := rfc2865.UserPassword_Set(request.Packet, []byte("password")); err != nil {
				t.Fatal(err)
			}
			writer := &capturedResponse{}
			server := &Server{}
			nas := net.ParseIP("203.0.113.7")
			if kind == "customer" {
				server.handleCustomerAuth(writer, request, &model.Customer{
					PPPoEUsername: "customer", PPPoEPassword: "password", Status: "active",
					IPAddress: "10.20.0.2", Package: &model.Package{Name: "Paket", IsActive: true},
				}, nas, authPAP, "juniper")
			} else {
				server.handleVoucherAuth(writer, request, &model.Voucher{
					Username: "voucher", Password: "password", Status: "active",
					Product: &model.VoucherProduct{Name: "Paket"},
				}, nas, authPAP, "juniper")
			}
			if writer.packet == nil || writer.packet.Code != radius.CodeAccessAccept {
				t.Fatalf("expected successful login, got %#v", writer.packet)
			}
			if got := rfc2869.AcctInterimInterval_Get(writer.packet); got != 300 {
				t.Fatalf("accounting interval = %d, want 300 seconds", got)
			}
		})
	}
}
