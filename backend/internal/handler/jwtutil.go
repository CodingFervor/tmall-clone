package handler

import "strconv"

// Hand-rolled base64url helpers (no external JWT library needed).

func encodeSeg(s string) string { return base64URL(s) }
func decodeSeg(s string) string { return base64URLDecode(s) }

func base64URL(s string) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	b := []byte(s)
	out := make([]byte, 0, len(b)*4/3+4)
	for i := 0; i < len(b); i += 3 {
		var n uint32
		var cnt int
		for j := 0; j < 3; j++ {
			if i+j < len(b) {
				n = n<<8 | uint32(b[i+j])
				cnt++
			} else {
				n <<= 8
			}
		}
		n <<= uint(8 * (3 - cnt))
		out = append(out, alphabet[(n>>18)&63])
		out = append(out, alphabet[(n>>12)&63])
		if cnt > 1 {
			out = append(out, alphabet[(n>>6)&63])
		}
		if cnt > 2 {
			out = append(out, alphabet[n&63])
		}
	}
	return string(out)
}

func base64URLDecode(s string) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	idx := make([]byte, 256)
	for i := range idx {
		idx[i] = 255
	}
	for i := 0; i < 64; i++ {
		idx[alphabet[i]] = byte(i)
	}
	out := make([]byte, 0, len(s)*3/4)
	var buf uint32
	var bits int
	for i := 0; i < len(s); i++ {
		v := idx[s[i]]
		if v == 255 {
			continue
		}
		buf = buf<<6 | uint32(v)
		bits += 6
		if bits >= 8 {
			bits -= 8
			out = append(out, byte(buf>>uint(bits)&0xFF))
		}
	}
	return string(out)
}

func extractInt(jsonStr, key string) int64 {
	needle := `"` + key + `":`
	i := indexOf(jsonStr, needle)
	if i < 0 {
		return 0
	}
	rest := jsonStr[i+len(needle):]
	end := 0
	for end < len(rest) {
		ch := rest[end]
		if (ch < '0' || ch > '9') && ch != '-' {
			break
		}
		end++
	}
	n, _ := strconv.ParseInt(rest[:end], 10, 64)
	return n
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
