package service

import (
	"context"
	"errors"
	"testing"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

func TestParseSplitRatio(t *testing.T) {
	cases := map[string]int{
		"1:2": 2, "1:4": 4, "1:8": 8, "1:16": 16, "1:64": 64,
		" 1:4 ": 4, "1 : 8": 8,
		"": 0, "1:4-custom": 0, "2:4": 0, "1:0": 0, "abc": 0, "10%/90%": 0,
	}
	for in, want := range cases {
		if got := parseSplitRatio(in); got != want {
			t.Errorf("parseSplitRatio(%q) = %d, want %d", in, got, want)
		}
	}
}

// fakeODPRepo: stub in-memory untuk uji validasi topologi. Method yang tidak
// di-override panic bila terpanggil (embedded nil interface).
type fakeODPRepo struct {
	repository.ODPRepository
	splitters map[string]*model.Splitter
	odpCount  map[string]int                    // splitterID -> keluaran terpakai ODP
	subCount  map[string]int                    // splitterID -> jumlah sub-splitter
	lineSeq   map[string]map[int]map[int]string // splitterID -> line -> sequence -> nama ODP
}

func (f *fakeODPRepo) FindSplitterByID(_ context.Context, _, id string) (*model.Splitter, error) {
	return f.splitters[id], nil
}
func (f *fakeODPRepo) CountSplitterOutputsUsed(_ context.Context, _, splitterID, _ string) (int, error) {
	return f.odpCount[splitterID], nil
}
func (f *fakeODPRepo) CountChildSplitters(_ context.Context, _, parentID, _ string) (int, error) {
	return f.subCount[parentID], nil
}
func (f *fakeODPRepo) FindODPBySplitterLineSeq(_ context.Context, _, splitterID string, line, sequence int, _ string) (*model.ODP, error) {
	if name, ok := f.lineSeq[splitterID][line][sequence]; ok {
		return &model.ODP{Name: name}, nil
	}
	return nil, nil
}
func (f *fakeODPRepo) LineOccupied(_ context.Context, _, splitterID string, line int, _ string) (bool, error) {
	return len(f.lineSeq[splitterID][line]) > 0, nil
}

func strp(s string) *string { return &s }
func intp(n int) *int       { return &n }

func TestValidateODPParent(t *testing.T) {
	repo := &fakeODPRepo{
		splitters: map[string]*model.Splitter{
			"odc1": {ID: "odc1", Name: "ODC-1", SplitterType: "1:4"},
		},
		odpCount: map[string]int{"odc1": 3},
		subCount: map[string]int{},
		lineSeq:  map[string]map[int]map[int]string{"odc1": {2: {1: "ODP-LAMA"}}},
	}
	svc := &ODPService{odpRepo: repo}
	ctx := context.Background()

	// Tanpa ODC: line dibuang, valid.
	odp := &model.ODP{TenantID: "t", SplitterLine: intp(3)}
	if err := svc.validateODPParent(ctx, odp); err != nil {
		t.Fatalf("tanpa ODC harus valid, dapat: %v", err)
	}
	if odp.SplitterLine != nil {
		t.Fatal("line harus di-nil-kan bila tidak ber-induk ODC")
	}

	// ODC tidak ada.
	odp = &model.ODP{TenantID: "t", SplitterID: strp("ghost")}
	if err := svc.validateODPParent(ctx, odp); !errors.Is(err, ErrSplitterNotFound) {
		t.Fatalf("ODC ghost harus ErrSplitterNotFound, dapat: %v", err)
	}

	// Keluaran ke-4 dari 1:4 (3 terpakai) masih boleh.
	odp = &model.ODP{TenantID: "t", SplitterID: strp("odc1"), SplitterLine: intp(1), Sequence: 1}
	if err := svc.validateODPParent(ctx, odp); err != nil {
		t.Fatalf("keluaran ke-4 dari 1:4 harus valid, dapat: %v", err)
	}

	// Kapasitas penuh: 4 dari 4 keluaran terpakai — line BARU ditolak.
	repo.odpCount["odc1"] = 4
	odp = &model.ODP{TenantID: "t", SplitterID: strp("odc1")}
	if err := svc.validateODPParent(ctx, odp); !errors.Is(err, ErrTopology) {
		t.Fatalf("ODC penuh harus ErrTopology, dapat: %v", err)
	}

	// Tapi MENYAMBUNG rantai line yang sudah ada tetap boleh walau penuh
	// (tidak memakan keluaran baru) — line 2 sudah berpenghuni urutan 1.
	odp = &model.ODP{TenantID: "t", SplitterID: strp("odc1"), SplitterLine: intp(2), Sequence: 2}
	if err := svc.validateODPParent(ctx, odp); err != nil {
		t.Fatalf("menyambung rantai line 2 harus valid, dapat: %v", err)
	}
	repo.odpCount["odc1"] = 3

	// Line di luar kapasitas.
	odp = &model.ODP{TenantID: "t", SplitterID: strp("odc1"), SplitterLine: intp(5)}
	if err := svc.validateODPParent(ctx, odp); !errors.Is(err, ErrTopology) {
		t.Fatalf("line 5 pada 1:4 harus ErrTopology, dapat: %v", err)
	}

	// Posisi (line, urutan) sudah dipakai ODP lain.
	odp = &model.ODP{TenantID: "t", SplitterID: strp("odc1"), SplitterLine: intp(2), Sequence: 1}
	if err := svc.validateODPParent(ctx, odp); !errors.Is(err, ErrTopology) {
		t.Fatalf("posisi line+urutan ganda harus ErrTopology, dapat: %v", err)
	}
}

func TestLineInputPower(t *testing.T) {
	// Rantai NET.id per line: tap 10% -> 20% -> 50% -> sisa (target seq 4).
	chain := []model.ODP{
		{ID: "o1", Sequence: 1, RatioPercent: 10},
		{ID: "o2", Sequence: 2, RatioPercent: 20},
		{ID: "o3", Sequence: 3, RatioPercent: 50},
		{ID: "o4", Sequence: 4, RatioPercent: 100},
	}
	// Pass-through: 90% (-0.46 dB), 80% (-0.97 dB), 50% (-3.01 dB) ≈ -4.44 dB.
	got := lineInputPower(0, chain, "o4", 4)
	if got < -4.6 || got > -4.3 {
		t.Fatalf("input power ODP4 harus ~-4.44 dB dari awal, dapat: %.2f", got)
	}
	// ODP pertama: belum ada pendahulu — power = start.
	if got := lineInputPower(-10, chain, "o1", 1); got != -10 {
		t.Fatalf("ODP urutan 1 harus terima power awal, dapat: %.2f", got)
	}
}

func TestValidateSplitterParent(t *testing.T) {
	// Rantai: a -> b -> c (a paling bawah, c root).
	repo := &fakeODPRepo{
		splitters: map[string]*model.Splitter{
			"a": {ID: "a", Name: "ODC-A", SplitterType: "1:4", ParentSplitterID: strp("b")},
			"b": {ID: "b", Name: "ODC-B", SplitterType: "1:4", ParentSplitterID: strp("c")},
			"c": {ID: "c", Name: "ODC-C", SplitterType: "1:2"},
		},
		odpCount: map[string]int{},
		subCount: map[string]int{"b": 1, "c": 1},
	}
	svc := &ODPService{odpRepo: repo}
	ctx := context.Background()

	// Format tipe salah.
	if err := svc.validateSplitterParent(ctx, "t", "4:1", nil, ""); !errors.Is(err, ErrTopology) {
		t.Fatalf("tipe 4:1 harus ErrTopology, dapat: %v", err)
	}

	// Tanpa induk: valid.
	if err := svc.validateSplitterParent(ctx, "t", "1:4", nil, ""); err != nil {
		t.Fatalf("tanpa induk harus valid, dapat: %v", err)
	}

	// Menginduk ke diri sendiri.
	if err := svc.validateSplitterParent(ctx, "t", "1:4", strp("a"), "a"); !errors.Is(err, ErrTopology) {
		t.Fatalf("induk diri sendiri harus ErrTopology, dapat: %v", err)
	}

	// Siklus: c mau menginduk ke a, padahal a -> b -> c.
	if err := svc.validateSplitterParent(ctx, "t", "1:2", strp("a"), "c"); !errors.Is(err, ErrTopology) {
		t.Fatalf("siklus harus ErrTopology, dapat: %v", err)
	}

	// Induk penuh: c bertipe 1:2 sudah punya 1 sub + 1 ODP.
	repo.odpCount["c"] = 1
	if err := svc.validateSplitterParent(ctx, "t", "1:8", strp("c"), ""); !errors.Is(err, ErrTopology) {
		t.Fatalf("induk penuh harus ErrTopology, dapat: %v", err)
	}

	// Induk masih longgar: b bertipe 1:4 punya 1 sub.
	if err := svc.validateSplitterParent(ctx, "t", "1:8", strp("b"), ""); err != nil {
		t.Fatalf("induk longgar harus valid, dapat: %v", err)
	}
}
