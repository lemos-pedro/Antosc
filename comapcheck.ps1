param(
  [Parameter(Mandatory)][string]$Host_,
  [int]$Port = 502, [int]$Unit = 1, [int]$Start = 0,
  [int]$Count = 10, [int]$Fc = 3, [int]$TimeoutMs = 8000
)

$c = New-Object System.Net.Sockets.TcpClient
$c.ReceiveTimeout = $TimeoutMs
$c.SendTimeout = $TimeoutMs
$c.Connect($Host_, $Port)
$s = $c.GetStream()

$req = [byte[]]@(
  0,1, 0,0, 0,6, $Unit, $Fc,
  [byte](($Start -shr 8) -band 255), [byte]($Start -band 255),
  [byte](($Count -shr 8) -band 255), [byte]($Count -band 255)
)
$s.Write($req, 0, $req.Length)

function Read-N($n) {
  $buf = New-Object byte[] $n; $t = 0
  while ($t -lt $n) {
    $r = $s.Read($buf, $t, $n - $t)
    if ($r -eq 0) { throw "ligação fechada" }
    $t += $r
  }
  ,$buf
}

$h = Read-N 9
if ($h[7] -band 0x80) { "Exceção Modbus, código $($h[8]) (1=função inválida, 2=endereço inválido)"; $c.Close(); return }
$d = Read-N $h[8]
for ($i = 0; $i -lt $d.Length; $i += 2) {
  $v = ($d[$i] * 256) + $d[$i+1]
  $sv = if ($v -gt 32767) { $v - 65536 } else { $v }
  "reg {0,5} = {1,6} (signed {2,6}, 0x{3:X4})" -f ($Start + $i/2), $v, $sv, $v
}
$c.Close()