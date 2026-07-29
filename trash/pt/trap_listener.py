# trap_listener.py
from pysnmp.entity import engine, config
from pysnmp.entity.rfc3413 import ntfrcv
from pysnmp.carrier.asyncio.dgram import udp

snmpEngine = engine.SnmpEngine()
config.addTransport(
    snmpEngine, udp.domainName,
    udp.UdpTransport().openServerMode(('0.0.0.0', 162))
)
config.addV1System(snmpEngine, 'my-area', 'TowercoreRead1')

def cbFun(snmpEngine, stateReference, contextEngineId, contextName,
          varBinds, cbCtx):
    print("Trap recebido!")
    for name, val in varBinds:
        print(f'{name.prettyPrint()} = {val.prettyPrint()}')

ntfrcv.NotificationReceiver(snmpEngine, cbFun)
snmpEngine.transportDispatcher.jobStarted(1)
try:
    snmpEngine.transportDispatcher.runDispatcher()
except:
    snmpEngine.transportDispatcher.closeDispatcher()