package fec

import (
	"encoding/binary"
)

// ═══════════════════════════════════════
// XOR-based Forward Error Correction
// ═══════════════════════════════════════
//
// ⚠️  STATUS: این ماژول هنوز در pipeline اصلی integrate نشده.
//     آماده استفاده هست ولی نیاز به wire در grouk/guarch داره.
//
// ایده: هر N بسته‌ی داده، ۱ بسته‌ی FEC بساز
// FEC = XOR(packet_1, packet_2, ..., packet_N)
// اگه ۱ بسته گم بشه، از بقیه + FEC بازسازی بشه
//
// ✅ H23: Length prefix: هر بسته قبل از XOR یک 2-byte length header میگیره
//         تا بسته‌های با طول مختلف درست بازسازی بشن
//
// فرمت داخلی هر بسته بعد از padding:
//   [2-byte origLen][original data][zero padding to maxLen]

// FECGroup — یک گروه FEC (encoder)
type FECGroup struct {
	GroupSize int      // تعداد بسته‌ها در هر گروه
	packets  [][]byte // بسته‌های padded شده (با length prefix)
	maxLen   int      // بزرگ‌ترین بسته‌ی padded
}

// NewFECGroup — ساخت گروه FEC جدید
func NewFECGroup(groupSize int) *FECGroup {
	if groupSize < 2 {
		groupSize = 4
	}
	return &FECGroup{
		GroupSize: groupSize,
		packets:  make([][]byte, 0, groupSize),
	}
}

// Add — اضافه کردن بسته به گروه
// اگه گروه پر شد، بسته‌ی FEC رو برمی‌گردونه
// ✅ H23: بسته‌ها با length-prefix ذخیره میشن برای بازسازی صحیح
func (fg *FECGroup) Add(data []byte) []byte {
	// ✅ H23: length prefix 2 byte
	padded := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(padded[0:2], uint16(len(data)))
	copy(padded[2:], data)

	fg.packets = append(fg.packets, padded)
	if len(padded) > fg.maxLen {
		fg.maxLen = len(padded)
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
// ✅ H23: همه بسته‌ها به maxLen pad میشن قبل از XOR
func (fg *FECGroup) generateFEC() []byte {
	result := make([]byte, fg.maxLen)

	for _, pkt := range fg.packets {
		for i := 0; i < len(pkt); i++ {
			result[i] ^= pkt[i]
		}
		// بایت‌های بعد از len(pkt) ضمنی صفرن → XOR تأثیری نداره
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
	packets  [][]byte // nil = گم شده (ذخیره با length prefix)
	fecData  []byte   // بسته‌ی FEC
	received int      // تعداد بسته‌های دریافت‌شده (بدون تکرار)
}

// NewFECDecoder — ساخت دیکدر
func NewFECDecoder(groupSize int) *FECDecoder {
	if groupSize < 2 {
		groupSize = 4
	}
	return &FECDecoder{
		GroupSize: groupSize,
		packets:  make([][]byte, groupSize),
	}
}

// AddPacket — اضافه کردن بسته‌ی داده (index = شماره در گروه)
// ✅ H23: length prefix اضافه میشه + شمارش تکراری اصلاح شد
func (fd *FECDecoder) AddPacket(index int, data []byte) {
	if index < 0 || index >= fd.GroupSize {
		return
	}

	// ✅ H23: length prefix (مثل encoder)
	padded := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(padded[0:2], uint16(len(data)))
	copy(padded[2:], data)

	// ✅ H23: فقط بسته‌های جدید شمرده بشن
	if fd.packets[index] == nil {
		fd.received++
	}
	fd.packets[index] = padded
}

// AddFEC — اضافه کردن بسته‌ی FEC
func (fd *FECDecoder) AddFEC(data []byte) {
	fd.fecData = make([]byte, len(data))
	copy(fd.fecData, data)
}

// CanRecover — آیا می‌تونه بسته‌ی گم‌شده رو بازسازی کنه؟
// ✅ H23: حالا از received استفاده میکنه
func (fd *FECDecoder) CanRecover() bool {
	if fd.fecData == nil {
		return false
	}
	// فقط ۱ بسته‌ی گم‌شده قابل بازسازیه
	// باید دقیقاً N-1 بسته داشته باشیم
	return fd.received == fd.GroupSize-1
}

// Recover — بازسازی بسته‌ی گم‌شده
// index بسته‌ی گم‌شده و داده‌ی بازسازی‌شده (بدون length prefix) رو برمی‌گردونه
// ✅ H23: length prefix بعد از XOR حذف میشه → طول صحیح
func (fd *FECDecoder) Recover() (int, []byte) {
	if !fd.CanRecover() {
		return -1, nil
	}

	// پیدا کردن بسته‌ی گم‌شده
	missingIdx := -1
	for i, p := range fd.packets {
		if p == nil {
			missingIdx = i
			break
		}
	}
	if missingIdx < 0 {
		return -1, nil
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

	// ✅ H23: استخراج طول اصلی از length prefix
	if len(result) < 2 {
		return missingIdx, nil
	}
	origLen := int(binary.BigEndian.Uint16(result[0:2]))
	if origLen+2 > len(result) {
		// best effort — ممکنه corrupted باشه
		return missingIdx, result[2:]
	}
	return missingIdx, result[2 : 2+origLen]
}

// Reset — ریست دیکدر
func (fd *FECDecoder) Reset() {
	for i := range fd.packets {
		fd.packets[i] = nil
	}
	fd.fecData = nil
	fd.received = 0
}
