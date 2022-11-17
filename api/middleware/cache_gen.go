package middleware

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *CacheKeyFlags) DecodeMsg(dc *msgp.Reader) (err error) {
	{
		var tmp uint
		tmp, err = dc.ReadUint()
		(*z) = CacheKeyFlags(tmp)
	}
	if err != nil {
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z CacheKeyFlags) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteUint(uint(z))
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z CacheKeyFlags) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendUint(o, uint(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *CacheKeyFlags) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var tmp uint
		tmp, bts, err = msgp.ReadUintBytes(bts)
		(*z) = CacheKeyFlags(tmp)
	}
	if err != nil {
		return
	}
	o = bts
	return
}

func (z CacheKeyFlags) Msgsize() (s int) {
	s = msgp.UintSize
	return
}

// DecodeMsg implements msgp.Decodable
func (z *CachedResponse) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var isz uint32
	isz, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for isz > 0 {
		isz--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "status":
			z.Status, err = dc.ReadInt()
			if err != nil {
				return
			}
		case "headers":
			var msz uint32
			msz, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Headers == nil && msz > 0 {
				z.Headers = make(map[string][]string, msz)
			} else if len(z.Headers) > 0 {
				for key, _ := range z.Headers {
					delete(z.Headers, key)
				}
			}
			for msz > 0 {
				msz--
				var xvk string
				var bzg []string
				xvk, err = dc.ReadString()
				if err != nil {
					return
				}
				var xsz uint32
				xsz, err = dc.ReadArrayHeader()
				if err != nil {
					return
				}
				if cap(bzg) >= int(xsz) {
					bzg = bzg[:xsz]
				} else {
					bzg = make([]string, xsz)
				}
				for bai := range bzg {
					bzg[bai], err = dc.ReadString()
					if err != nil {
						return
					}
				}
				z.Headers[xvk] = bzg
			}
		case "body":
			z.Body, err = dc.ReadBytes(z.Body)
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *CachedResponse) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "status"
	err = en.Append(0x83, 0xa6, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteInt(z.Status)
	if err != nil {
		return
	}
	// write "headers"
	err = en.Append(0xa7, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteMapHeader(uint32(len(z.Headers)))
	if err != nil {
		return
	}
	for xvk, bzg := range z.Headers {
		err = en.WriteString(xvk)
		if err != nil {
			return
		}
		err = en.WriteArrayHeader(uint32(len(bzg)))
		if err != nil {
			return
		}
		for bai := range bzg {
			err = en.WriteString(bzg[bai])
			if err != nil {
				return
			}
		}
	}
	// write "body"
	err = en.Append(0xa4, 0x62, 0x6f, 0x64, 0x79)
	if err != nil {
		return err
	}
	err = en.WriteBytes(z.Body)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *CachedResponse) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "status"
	o = append(o, 0x83, 0xa6, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73)
	o = msgp.AppendInt(o, z.Status)
	// string "headers"
	o = append(o, 0xa7, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73)
	o = msgp.AppendMapHeader(o, uint32(len(z.Headers)))
	for xvk, bzg := range z.Headers {
		o = msgp.AppendString(o, xvk)
		o = msgp.AppendArrayHeader(o, uint32(len(bzg)))
		for bai := range bzg {
			o = msgp.AppendString(o, bzg[bai])
		}
	}
	// string "body"
	o = append(o, 0xa4, 0x62, 0x6f, 0x64, 0x79)
	o = msgp.AppendBytes(o, z.Body)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *CachedResponse) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var isz uint32
	isz, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for isz > 0 {
		isz--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "status":
			z.Status, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				return
			}
		case "headers":
			var msz uint32
			msz, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Headers == nil && msz > 0 {
				z.Headers = make(map[string][]string, msz)
			} else if len(z.Headers) > 0 {
				for key, _ := range z.Headers {
					delete(z.Headers, key)
				}
			}
			for msz > 0 {
				var xvk string
				var bzg []string
				msz--
				xvk, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				var xsz uint32
				xsz, bts, err = msgp.ReadArrayHeaderBytes(bts)
				if err != nil {
					return
				}
				if cap(bzg) >= int(xsz) {
					bzg = bzg[:xsz]
				} else {
					bzg = make([]string, xsz)
				}
				for bai := range bzg {
					bzg[bai], bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						return
					}
				}
				z.Headers[xvk] = bzg
			}
		case "body":
			z.Body, bts, err = msgp.ReadBytesBytes(bts, z.Body)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

func (z *CachedResponse) Msgsize() (s int) {
	s = 1 + 7 + msgp.IntSize + 8 + msgp.MapHeaderSize
	if z.Headers != nil {
		for xvk, bzg := range z.Headers {
			_ = bzg
			s += msgp.StringPrefixSize + len(xvk) + msgp.ArrayHeaderSize
			for bai := range bzg {
				s += msgp.StringPrefixSize + len(bzg[bai])
			}
		}
	}
	s += 5 + msgp.BytesPrefixSize + len(z.Body)
	return
}
