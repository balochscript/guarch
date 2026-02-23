package fec

// ═══════════════════════════════════════
// XOR-based Forward Error Correction
// ═══════════════════════════════════════
//
// ایده: هر N بسته‌ی داده، ۱ بسته‌ی FEC بساز
// FEC = XOR(packet_1, packet_2, ..., packet_N)
// اگه ۱ بسته گم بشه، از بقیه + FEC بازسازی بشه
//
// مثال (N=3):
//   ارسال: [P1] [P2] [P3] [FEC = P1⊕P2⊕P3]
//   گم:    [P1] [ ] [P3] [FEC]
//   بازسازی: P2 = P1 ⊕ P3 ⊕ FEC

// FECGroup — یک گروه FEC
type FECGroup struct {
	GroupSize  int      // تعداد بسته‌ها در هر گروه
	packets   [][]byte // بسته‌های جمع‌آوری‌شده
	maxLen    int      // بزرگ‌ترین بسته
}

// NewFECGroup — ساخت گروه FEC جدید
func NewFECGroup(groupSize int) *FECGroup {
	if groupSize < 2 {
		groupSize = 4 // پیش‌فرض: هر ۴ بسته ۱ FEC
	}
	return &FECGroup{
		GroupSize: groupSize,
		packets:  make([][]byte, 0, groupSize),
	}
}

// Add — اضافه کردن بسته به گروه
// اگه گروه پر شد، بسته‌ی FEC رو برمی‌گردونه
func (fg *FECGroup) Add(data []byte) []byte {
	// کپی
	cp := make([]byte, len(data))
	copy(cp, data)
	fg.packets = append(fg.packets, cp)

	if len(cp) > fg.maxLen {
		fg.maxLen = len(cp)
	}

	// گروه پر شد؟
	if len(fg.packets) >= fg.GroupSize {
		fec := fg.generateFEC()
		fg.Reset()
		return fec
	}

	return nil
}

// generateFEC — تولید بسته‌ی FEC (XOR همه)
func (fg *FECGroup) generateFEC() []byte {
	result := make([]byte, fg.maxLen)

	for _, pkt := range fg.packets {
		for i := 0; i < len(pkt); i++ {
			result[i] ^= pkt[i]
		}
	}

	return result
}

// Reset — ریست گروه
func (fg *FECGroup) Reset() {
	fg.packets = fg.packets[:0]
	fg.maxLen = 0
}

// ═══════════════════════════════════════
// FEC Decoder — بازسازی بسته‌ی گم‌شده
// ═══════════════════════════════════════

// FECDecoder — دیکدر FEC
type FECDecoder struct {
	GroupSize int
	packets  [][]byte  // nil = گم شده
	fecData  []byte    // بسته‌ی FEC
	received int       // تعداد دریافت‌شده
}

// NewFECDecoder — ساخت دیکدر
func NewFECDecoder(groupSize int) *FECDecoder {
	return &FECDecoder{
		GroupSize: groupSize,
		packets:  make([][]byte, groupSize),
	}
}

// AddPacket — اضافه کردن بسته‌ی داده (index = شماره در گروه)
func (fd *FECDecoder) AddPacket(index int, data []byte) {
	if index >= 0 && index < fd.GroupSize {
		cp := make([]byte, len(data))
		copy(cp, data)
		fd.packets[index] = cp
		fd.received++
	}
}

// AddFEC — اضافه کردن بسته‌ی FEC
func (fd *FECDecoder) AddFEC(data []byte) {
	fd.fecData = make([]byte, len(data))
	copy(fd.fecData, data)
}

// CanRecover — آیا می‌تونه بسته‌ی گم‌شده رو بازسازی کنه؟
func (fd *FECDecoder) CanRecover() bool {
	if fd.fecData == nil {
		return false
	}
	// فقط ۱ بسته‌ی گم‌شده قابل بازسازیه
	missing := 0
	for _, p := range fd.packets {
		if p == nil {
			missing++
		}
	}
	return missing == 1
}

// Recover — بازسازی بسته‌ی گم‌شده
// index بسته‌ی گم‌شده و داده‌ی بازسازی‌شده رو برمی‌گردونه
func (fd *FECDecoder) Recover() (int, []byte) {
	if !fd.CanRecover() {
		return -1, nil
	}

	missingIdx := -1
	for i, p := range fd.packets {
		if p == nil {
			missingIdx = i
			break
		}
	}

	// بازسازی: missing = FEC ⊕ all_other_packets
	result := make([]byte, len(fd.fecData))
	copy(result, fd.fecData)

	for i, pkt := range fd.packets {
		if i == missingIdx || pkt == nil {
			continue
		}
		for j := 0; j < len(pkt) && j < len(result); j++ {
			result[j] ^= pkt[j]
		}
	}

	return missingIdx, result
}

// Reset — ریست دیکدر
func (fd *FECDecoder) Reset() {
	for i := range fd.packets {
		fd.packets[i] = nil
	}
	fd.fecData = nil
	fd.received = 0
}
