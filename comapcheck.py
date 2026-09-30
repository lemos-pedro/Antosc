import socket, struct, sys, argparse

p = argparse.ArgumentParser()
p.add_argument("--host", required=True)
p.add_argument("--port", type=int, default=502)
p.add_argument("--unit", type=int, default=1)
p.add_argument("--start", type=int, default=0)
p.add_argument("--count", type=int, default=10)
p.add_argument("--fc", type=int, default=3, help="3=holding, 4=input")
p.add_argument("--timeout", type=float, default=8)
a = p.parse_args()

def recv_all(s, n):
    buf = b""
    while len(buf) < n:
        chunk = s.recv(n - len(buf))
        if not chunk:
            raise ConnectionError("ligação fechada pelo controlador")
        buf += chunk
    return buf

try:
    s = socket.create_connection((a.host, a.port), timeout=a.timeout)
    s.settimeout(a.timeout)
    s.sendall(struct.pack(">HHHBBHH", 1, 0, 6, a.unit, a.fc, a.start, a.count))
    hdr = recv_all(s, 9)
    if hdr[7] & 0x80:
        print(f"Exceção Modbus, código {hdr[8]} (1=função inválida, 2=endereço inválido)")
        sys.exit(1)
    data = recv_all(s, hdr[8])
    for i in range(0, len(data), 2):
        v = struct.unpack(">H", data[i:i+2])[0]
        sv = struct.unpack(">h", data[i:i+2])[0]
        print(f"reg {a.start + i//2:5d} = {v:6d} (signed {sv:6d}, 0x{v:04X})")
except Exception as e:
    print("ERRO:", e)
    sys.exit(1)