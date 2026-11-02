package tradecalendar

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

const (
	// base64Alphabet 是站点使用的字符表，与标准 Base64 一致。
	base64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	// epochDayBase 是基准日期相对 1970-01-01 的天数，对应 1990-12-19（上交所开市日）。
	epochDayBase = 7657
)

// datelistPattern 匹配响应中的日期编码串。
var datelistPattern = regexp.MustCompile(`var\s+datelist="([^"]+)"`)

var (
	// ErrEncoding 表示日期编码本身非法，例如字符不在字符表内或数据提前结束。
	ErrEncoding = errors.New("交易日历编码非法")
	// ErrChecksum 表示头部校验和不匹配，说明响应内容已损坏或格式已变更。
	ErrChecksum = errors.New("交易日历校验和不匹配")
)

// Parse 解析新浪交易日历文本，返回解码后的日期列表。
//
// 入参是完整的响应文本（含 `var datelist="..."` 声明）。文本中找不到该声明时返回
// ErrNoData；编码非法返回 ErrEncoding；头部校验和不匹配返回 ErrChecksum。
func Parse(body string) (*TradeCalendar, error) {
	m := datelistPattern.FindStringSubmatch(body)
	if len(m) < 2 || m[1] == "" {
		return nil, ErrNoData
	}
	dates, err := decode(m[1])
	if err != nil {
		return nil, err
	}
	return &TradeCalendar{Dates: dates}, nil
}

// decode 把编码串还原为按日期升序的列表。
//
// 编码规则（实测确认，与站点前端一致）：
//
//   - 每个字符按 Base64 字符表映射为 6 位，按低位在前的顺序串成位流；
//   - 依次读出 12 位与 6 位作为头部，第二个字段与 63 异或得到校验和，只接受 0 或 1；
//   - 再读两个 18 位整数，分别作为起始游标与结束游标；
//   - 游标按「加一天、遇到周末折叠到周一」推进，每推进一天先判断是否需要读入新一段
//     的重复次数：先读 1 位标志，为 1 时再读变长符号位调整游程长度，随后按
//     游程长度 ×3 位读入本段的重复天数。
func decode(encoded string) ([]time.Time, error) {
	d := newDecoder(encoded)
	if d.empty() {
		return nil, fmt.Errorf("%w: 编码为空", ErrEncoding)
	}

	header, err := d.readBits([]int{12, 6})
	if err != nil {
		return nil, err
	}
	if checksum := header[1] ^ 63; checksum > 1 {
		return nil, fmt.Errorf("%w: 校验和=%d", ErrChecksum, checksum)
	}

	start, err := d.readBits([]int{18})
	if err != nil {
		return nil, err
	}
	end, err := d.readBits([]int{18})
	if err != nil {
		return nil, err
	}

	day := start[0] - 1
	limit := end[0]
	if limit < day {
		return nil, fmt.Errorf("%w: 起止游标异常", ErrEncoding)
	}

	dates := make([]time.Time, 0, limit-day)
	remaining := -1
	runLength := 0
	for day < limit {
		date, next := advanceDay(day)
		day = next

		if remaining <= 0 {
			if d.nextBit() {
				// 游程长度是累计量，可为负；负值时本段长度为 0，不读入额外的位。
				runLength += d.readSign()
			}
			value, err := d.readBits([]int{runLength * 3})
			if err != nil {
				return nil, err
			}
			remaining = value[0] + 1
			if len(dates) == 0 {
				dates = append(dates, date)
				remaining--
			}
		} else {
			dates = append(dates, date)
		}
		remaining--
	}
	return dates, nil
}

// advanceDay 把游标推进一天，周末折叠到周一，返回对应日期与新游标。
func advanceDay(day int) (time.Time, int) {
	day++
	if r := day % 7; r == 3 || r == 4 {
		day += 5 - r
	}
	return epoch().AddDate(0, 0, epochDayBase+day), day
}

// epoch 是日期换算的基准，固定用 UTC，避免受本地时区影响。
func epoch() time.Time {
	return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
}

// decoder 是一个按 6 位一组读取的位流。
type decoder struct {
	bits []int
	pos  int
	off  int
}

// newDecoder 把编码串转成 6 位值序列，遇到非法字符时返回只含已解析部分的结果。
func newDecoder(encoded string) *decoder {
	bits := make([]int, 0, len(encoded))
	for _, r := range encoded {
		idx := indexOf(r)
		if idx < 0 {
			break
		}
		bits = append(bits, idx)
	}
	return &decoder{bits: bits}
}

// indexOf 返回字符在 Base64 字符表中的位置，未找到返回 -1。
func indexOf(r rune) int {
	for i := 0; i < len(base64Alphabet); i++ {
		if rune(base64Alphabet[i]) == r {
			return i
		}
	}
	return -1
}

// empty 表示位流为空。
func (d *decoder) empty() bool { return len(d.bits) == 0 }

// exhausted 表示已读到位流末尾。
func (d *decoder) exhausted() bool { return d.pos >= len(d.bits) }

// nextBit 读 1 位，末尾返回 false。
func (d *decoder) nextBit() bool {
	if d.exhausted() {
		return false
	}
	bit := d.bits[d.pos]&(1<<d.off) != 0
	d.off++
	if d.off >= 6 {
		d.off -= 6
		d.pos++
	}
	return bit
}

// readSign 读变长的符号位，返回游程长度的增量。
//
// 第一个位表示方向：1 为正向、0 为反向；随后的连续 1 的个数决定增量大小。
func (d *decoder) readSign() int {
	positive := d.nextBit()
	count := 1
	for d.nextBit() {
		count++
	}
	if positive {
		return count
	}
	return -count
}

// readBits 依次读取若干定长字段，长度为 0 时该字段记为 0。
func (d *decoder) readBits(lengths []int) ([]int, error) {
	values := make([]int, 0, len(lengths))
	for _, length := range lengths {
		// 长度为 0 或负数时该字段记为 0，且不消耗位。
		if length <= 0 {
			values = append(values, 0)
			continue
		}
		value, err := d.readValue(length)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// readValue 读取 length 位，低位在前。
//
// 超过 30 位的字段拆成 30 位与剩余部分拼接，避免位移溢出。
func (d *decoder) readValue(length int) (int, error) {
	if length <= 30 {
		return d.readShort(length)
	}
	parts, err := d.readBits([]int{30, length - 30})
	if err != nil {
		return 0, err
	}
	return parts[0] + parts[1]<<30, nil
}

// readShort 读取不超过 30 位的字段。
func (d *decoder) readShort(length int) (int, error) {
	value := 0
	remaining := length
	for remaining > 0 {
		if d.exhausted() {
			return 0, fmt.Errorf("%w: 数据提前结束", ErrEncoding)
		}
		available := 6 - d.off
		take := min(remaining, available)
		mask := 1<<take - 1
		value |= (d.bits[d.pos] >> d.off & mask) << (length - remaining)
		d.off += take
		if d.off >= 6 {
			d.off -= 6
			d.pos++
		}
		remaining -= take
	}
	return value, nil
}
