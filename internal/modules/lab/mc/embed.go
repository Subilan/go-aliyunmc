package mc

import _ "embed"

//go:embed en_us.json
var langEnJSON []byte

//go:embed zh_cn.json
var langZhJSON []byte
