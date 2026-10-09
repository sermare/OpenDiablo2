package d2realm

import "github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"

func tunnelFor(typ byte, body []byte) [][]byte { return d2gs.Tunnel(d2gs.CtlTunnel, typ, body) }
