// © 2016 Steve McCoy under the MIT license. See LICENSE for details.

package ogg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strconv"
	"strings"
)

// A Decoder decodes an ogg stream page-by-page with its Decode method.
type Decoder struct {
	// buffer for packet lengths, to avoid allocating (mss is also the max per page)
	lenbuf [mss]int
	r      io.Reader
	buf    [maxPageSize]byte
}

// NewDecoder creates an ogg Decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// A Page represents a logical ogg page.
type Page struct {
	// Type is a bitmask of COP, BOS, and/or EOS.
	Type byte
	// Serial is the bitstream serial number.
	Serial uint32
	// Granule is the granule position, whose meaning is dependent on the encapsulated codec.
	Granule int64
	// Packets are the raw packet data.
	// If Type & COP != 0, the first element is
	// a continuation of the previous page's last packet.
	Packets [][]byte
}

// ErrBadSegs is the error used when trying to decode a page with a segment table size less than 1.
var ErrBadSegs = errors.New("invalid segment table size")

// ErrBadCrc is the error used when an ogg page's CRC field does not match the CRC calculated by the Decoder.
type ErrBadCrc struct {
	Found    uint32
	Expected uint32
}

func (bc ErrBadCrc) Error() string {
	return "invalid crc in packet: got " + strconv.FormatInt(int64(bc.Found), 16) +
		", expected " + strconv.FormatInt(int64(bc.Expected), 16)
}

var oggs = []byte{'O', 'g', 'g', 'S'}

// Decode reads from d's Reader to the next ogg page, then returns the decoded Page or an error.
// The error may be io.EOF if that's what the Reader returned.
//
// The buffer underlying the returned Page's Packets' bytes is owned by the Decoder.
// It may be overwritten by subsequent calls to Decode.
//
// It is safe to call Decode concurrently on distinct Decoders if their Readers are distinct.
// Otherwise, the behavior is undefined.
func (d *Decoder) Decode() (Page, error) {
	hbuf := d.buf[0:headsz]
	b := 0
	for {
		_, err := io.ReadFull(d.r, hbuf[b:])
		if err != nil {
			return Page{}, err
		}

		i := bytes.Index(hbuf, oggs)
		if i == 0 {
			break
		}

		if i < 0 {
			const n = headsz
			if hbuf[n-1] == 'O' {
				i = n - 1
			} else if hbuf[n-2] == 'O' && hbuf[n-1] == 'g' {
				i = n - 2
			} else if hbuf[n-3] == 'O' && hbuf[n-2] == 'g' && hbuf[n-1] == 'g' {
				i = n - 3
			}
		}

		if i > 0 {
			b = copy(hbuf, hbuf[i:])
		}
	}

	var h pageHeader
	_ = binary.Read(bytes.NewBuffer(hbuf), byteOrder, &h)

	if h.Nsegs < 1 {
		return Page{}, ErrBadSegs
	}

	nsegs := int(h.Nsegs)
	segtbl := d.buf[headsz : headsz+nsegs]
	_, err := io.ReadFull(d.r, segtbl)
	if err != nil {
		return Page{}, err
	}

	// A page can contain multiple packets; record their lengths from the table
	// now and slice up the payload after reading it.
	// I'm inclined to limit the Read calls this way,
	// but it's possible it isn't worth the annoyance of iterating twice
	packetlens := d.lenbuf[0:0]
	payloadlen := 0
	more := false
	for _, l := range segtbl {
		if more {
			packetlens[len(packetlens)-1] += int(l)
		} else {
			packetlens = append(packetlens, int(l))
		}

		more = l == mss
		payloadlen += int(l)
	}

	payload := d.buf[headsz+nsegs : headsz+nsegs+payloadlen]
	_, err = io.ReadFull(d.r, payload)
	if err != nil {
		return Page{}, err
	}

	page := d.buf[0 : headsz+nsegs+payloadlen]
	// Clear out existing crc before calculating it
	page[22] = 0
	page[23] = 0
	page[24] = 0
	page[25] = 0
	crc := crc32(page)
	if crc != h.Crc {
		return Page{}, ErrBadCrc{h.Crc, crc}
	}

	packets := make([][]byte, len(packetlens))
	s := 0
	for i, l := range packetlens {
		packets[i] = payload[s : s+l]
		s += l
	}

	return Page{h.HeaderType, h.Serial, h.Granule, packets}, nil
}

var (
	ErrBadIDHeader      = errors.New("invalid id header packets")
	ErrBadCommentHeader = errors.New("invalid comment header packets")
)

// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      'O'      |      'p'      |      'u'      |      's'      |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      'H'      |      'e'      |      'a'      |      'd'      |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |  Version = 1  | Channel Count |           Pre-skip            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                     Input Sample Rate (Hz)                    |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |   Output Gain (Q7.8 in dB)    | Mapping Family|               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+               :
// |                                                               |
// :               Optional Channel Mapping Table...               :
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

type IDHeader struct {
	Version              uint8
	OutputChannelCount   uint8
	PreSkip              uint16
	InputSampleRate      uint32
	OutputGainQ7_8       int16
	ChannelMappingFamily uint8 // 0 1 255
	// ChannelMapping
}

func (idh *IDHeader) Decode(packets [][]byte) error {
	if len(packets) == 0 {
		return ErrBadIDHeader
	}
	if len(packets[0]) < 8+1+1+2+4+2+1 {
		return ErrBadIDHeader
	}
	if !bytes.HasPrefix(packets[0], []byte("OpusHead")) {
		return ErrBadIDHeader
	}

	idh.Version = packets[0][8]
	idh.OutputChannelCount = packets[0][9]

	idh.PreSkip = binary.LittleEndian.Uint16(packets[0][10:])
	idh.InputSampleRate = binary.LittleEndian.Uint32(packets[0][12:])

	err := binary.Read(bytes.NewReader(packets[0][16:]), binary.LittleEndian, &idh.OutputGainQ7_8)
	if err != nil {
		return err
	}

	idh.ChannelMappingFamily = packets[0][18]
	return nil
}

func (idh *IDHeader) Encode() ([][]byte, error) {
	buf := bytes.NewBuffer(nil)
	_, err := buf.WriteString("OpusHead")
	if err != nil {
		return nil, err
	}

	err = buf.WriteByte(byte(idh.Version))
	if err != nil {
		return nil, err
	}

	err = buf.WriteByte(byte(idh.OutputChannelCount))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, uint16(idh.PreSkip))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, uint32(idh.InputSampleRate))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, uint16(idh.OutputGainQ7_8))
	if err != nil {
		return nil, err
	}

	err = buf.WriteByte(byte(idh.ChannelMappingFamily))
	if err != nil {
		return nil, err
	}

	return [][]byte{buf.Bytes()}, nil
}

//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      'O'      |      'p'      |      'u'      |      's'      |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      'T'      |      'a'      |      'g'      |      's'      |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                     Vendor String Length                      |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// :                        Vendor String...                       :
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                   User Comment List Length                    |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 User Comment #0 String Length                 |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// :                   User Comment #0 String...                   :
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 User Comment #1 String Length                 |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// :                                                               :

type CommentHeader struct {
	VendorString    string
	UserCommentList map[string]string
}

func (cmh *CommentHeader) Decode(packets [][]byte) error {
	if len(packets) == 0 {
		return ErrBadCommentHeader
	}
	if len(packets[0]) < 8+4+4 {
		return ErrBadCommentHeader
	}
	if !bytes.HasPrefix(packets[0], []byte("OpusTags")) {
		return ErrBadCommentHeader
	}

	var allPackets []byte
	for i := range packets {
		allPackets = append(allPackets, packets[i]...)
	}

	r := bytes.NewReader(allPackets[8:])

	var vendorStringLen uint32
	err := binary.Read(r, binary.LittleEndian, &vendorStringLen)
	if err != nil {
		return err
	}

	if vendorStringLen > 0 {
		temp := make([]byte, vendorStringLen)
		_, err = io.ReadFull(r, temp)
		if err != nil {
			return err
		}

		cmh.VendorString = string(temp)
	}

	var userCommentListLen uint32
	err = binary.Read(r, binary.LittleEndian, &userCommentListLen)
	if err != nil {
		return err
	}

	if userCommentListLen > 0 {
		cmh.UserCommentList = make(map[string]string)

		for range userCommentListLen {

			var userCommentLen uint32
			err := binary.Read(r, binary.LittleEndian, &userCommentLen)
			if err != nil {
				return err
			}

			if userCommentLen == 0 {
				return errors.New("user comment must be non-empty")
			}

			temp := make([]byte, userCommentLen)
			_, err = io.ReadFull(r, temp)
			if err != nil {
				return err
			}

			s := string(temp)

			arr := strings.SplitN(s, "=", 2)

			switch len(arr) {
			case 1:
				cmh.UserCommentList[arr[0]] = ""
			case 2:
				cmh.UserCommentList[arr[0]] = arr[1]
			}

		}
	}

	return nil
}

func (cmh *CommentHeader) Encode() ([][]byte, error) {
	buf := bytes.NewBuffer(nil)
	_, err := buf.WriteString("OpusTags")
	if err != nil {
		return nil, err
	}

	vendorStringLen := len(cmh.VendorString)

	err = binary.Write(buf, binary.LittleEndian, uint32(vendorStringLen))
	if err != nil {
		return nil, err
	}

	if vendorStringLen > 0 {
		_, err = buf.WriteString(cmh.VendorString)
		if err != nil {
			return nil, err
		}
	}

	userCommentsLen := len(cmh.UserCommentList)

	err = binary.Write(buf, binary.LittleEndian, uint32(userCommentsLen))
	if err != nil {
		return nil, err
	}

	for k, v := range cmh.UserCommentList {
		comment := k + "=" + v
		commentLen := len(comment)

		err = binary.Write(buf, binary.LittleEndian, uint32(commentLen))
		if err != nil {
			return nil, err
		}

		if commentLen > 0 {
			_, err = buf.WriteString(comment)
			if err != nil {
				return nil, err
			}
		}

	}

	return [][]byte{buf.Bytes()}, nil
}

func (d *Decoder) ParseIDHeader() (*IDHeader, error) {
	p, err := d.Decode()
	if err != nil {
		return nil, err
	}

	if p.Type&BOS != BOS {
		return nil, errors.New("can't read BOS")
	}

	var idh = &IDHeader{}
	err = idh.Decode(p.Packets)
	if err != nil {
		return nil, err
	}

	return idh, nil
}
