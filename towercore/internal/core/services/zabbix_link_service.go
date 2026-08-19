package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"towercore/internal/adapters/zabbix"
	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/logger"
)

type ZabbixLinkSyncService struct {
	client *zabbix.Client
	links interfaces.NetworkLinkRepository
	log *logger.Logger
}

func NewZabbixLinkSyncService(client *zabbix.Client, links interfaces.NetworkLinkRepository, log *logger.Logger) *ZabbixLinkSyncService { return &ZabbixLinkSyncService{client:client,links:links,log:log} }

type ZabbixSyncResult struct { Hosts int `json:"hosts"`; LinksCreated int `json:"links_created"`; LinksUpdated int `json:"links_updated"`; Snapshots int `json:"snapshots"`; Errors []string `json:"errors,omitempty"` }

func (s *ZabbixLinkSyncService) Inspect(ctx context.Context, searches []string) (map[string][]zabbix.Item, error) {
	_ = ctx
	if len(searches) == 0 { return nil, fmt.Errorf("host_search is required to avoid importing every Zabbix host") }
	if err := s.client.Login(); err != nil { return nil, err }
	hosts, err := s.client.GetHostsByNameFilter(searches); if err != nil{return nil,err}
	result := make(map[string][]zabbix.Item,len(hosts)); for _, host := range hosts { items, e := s.client.GetItemsForHost(host.HostID); if e != nil{return nil,e}; result[host.Name]=items }; return result,nil
}

func (s *ZabbixLinkSyncService) Sync(ctx context.Context, searches []string) (ZabbixSyncResult, error) {
	var out ZabbixSyncResult
	if len(searches) == 0 { return out, fmt.Errorf("host_search is required to avoid importing every Zabbix host") }
	if err := s.client.Login(); err != nil{return out,err}
	hosts, err := s.client.GetHostsByNameFilter(searches); if err != nil{return out,err}; out.Hosts=len(hosts)
	for _, host := range hosts {
		items, e := s.client.GetItemsForHost(host.HostID); if e != nil { out.Errors=append(out.Errors,fmt.Sprintf("%s: %v",host.Name,e)); continue }
		groups := groupZabbixItems(items)
		for ifIndex, group := range groups {
			if err := s.syncGroup(ctx, host, ifIndex, group, &out); err != nil { out.Errors=append(out.Errors,fmt.Sprintf("%s[%d]: %v",host.Name,ifIndex,err)) }
		}
	}
	return out,nil
}

func (s *ZabbixLinkSyncService) syncGroup(ctx context.Context, host zabbix.Host, ifIndex int, items []zabbix.Item, out *ZabbixSyncResult) error {
	routerIP := host.Host; for _, in := range host.Interfaces { if in.IP != "" { routerIP=in.IP; break } }
	values := make(map[string]string); var clock time.Time
	for _, item := range items { values[itemMetricName(item)] = item.LastValue; if c,err:=strconv.ParseInt(item.LastClock,10,64);err==nil && c>0 { clock=time.Unix(c,0).UTC() } }
	if clock.IsZero(){clock=time.Now().UTC()}
	link, err := s.links.GetByRouterAndIfIndex(ctx,routerIP,ifIndex); if err != nil{return err}
	if link == nil { link=&domain.NetworkLink{RouterHost:host.Name,RouterIP:routerIP,IfIndex:ifIndex,MediaType:"fiber",Operator:host.Name} }
	link.RouterHost=host.Name; link.RouterIP=routerIP; link.IfIndex=ifIndex
	link.IfDescr=first(values,"ifdescr","descr"); link.IfAlias=first(values,"ifalias","alias")
	if link.IfDescr=="" {link.IfDescr=fmt.Sprintf("ifIndex %d",ifIndex)}; if link.Name=="" {link.Name=link.IfAlias; if link.Name==""{link.Name=link.IfDescr}}
	link.NominalCapacityMb=parseCapacity(values)
	if link.LinkID=="" {if err:=s.links.Create(ctx,link);err!=nil{return err};out.LinksCreated++} else {if err:=s.links.Update(ctx,link);err!=nil{return err};out.LinksUpdated++}
	snapshot:=&domain.LinkMetricSnapshot{LinkID:link.LinkID,CollectedAt:clock,OperStatus:status(values["operstatus"]),AdminStatus:status(values["adminstatus"]),InOctets:uintValue(values,"in_octets"),OutOctets:uintValue(values,"out_octets"),InErrors:uintValue(values,"in_errors"),OutErrors:uintValue(values,"out_errors"),InDiscards:uintValue(values,"in_discards"),OutDiscards:uintValue(values,"out_discards"),SpeedMb:parseCapacity(values)}
	if snapshot.OperStatus=="" { snapshot.OperStatus=domain.LinkStatusUnknown }; if snapshot.AdminStatus=="" {snapshot.AdminStatus=domain.LinkStatusUnknown}
	if err:=s.links.SaveMetricSnapshot(ctx,snapshot);err!=nil{return err}; out.Snapshots++; return nil
}

var ifIndexPattern=regexp.MustCompile(`(?i)(?:\[|[.,:])(\d+)\]?$`)
func groupZabbixItems(items []zabbix.Item) map[int][]zabbix.Item { out:=map[int][]zabbix.Item{}; for _,item:=range items { n:=itemMetricName(item); if !isLinkMetric(n){continue}; m:=ifIndexPattern.FindStringSubmatch(item.Key); if len(m)==0 {m=ifIndexPattern.FindStringSubmatch(item.Name)}; if len(m)==0{continue}; i,_:=strconv.Atoi(m[1]); out[i]=append(out[i],item) }; return out }
func itemMetricName(i zabbix.Item) string { text:=strings.ToLower(i.Key+" "+i.Name); switch {case strings.Contains(text,"operstatus"):return "operstatus";case strings.Contains(text,"adminstatus"):return "adminstatus";case strings.Contains(text,"hcin")||strings.Contains(text,"in_octets")||strings.Contains(text,"inoctets"):return "in_octets";case strings.Contains(text,"hcout")||strings.Contains(text,"out_octets")||strings.Contains(text,"outoctets"):return "out_octets";case strings.Contains(text,"inerrors"):return "in_errors";case strings.Contains(text,"outerrors"):return "out_errors";case strings.Contains(text,"indiscards"):return "in_discards";case strings.Contains(text,"outdiscards"):return "out_discards";case strings.Contains(text,"highspeed")||strings.Contains(text,"speed"):return "speed";case strings.Contains(text,"ifalias"):return "ifalias";case strings.Contains(text,"ifdescr"):return "ifdescr"}; return "" }
func isLinkMetric(n string) bool{return n!=""}
func first(m map[string]string, keys ...string)string{for _,k:=range keys{if m[k]!=""{return m[k]}};return ""}
func parseCapacity(m map[string]string)int64{v:=first(m,"speed");f,_:=strconv.ParseFloat(v,64);if f<0{return 0};if f>1000000{return int64(f/1000000)};return int64(f)}
func uintValue(m map[string]string,k string)uint64{v,_:=strconv.ParseUint(first(m,k),10,64);return v}
func status(v string)domain.LinkOperStatus{switch strings.ToLower(strings.TrimSpace(v)){case"1","up":return domain.LinkStatusUp;case"2","down":return domain.LinkStatusDown;case"3","testing":return domain.LinkStatusTesting;case"5","dormant":return domain.LinkStatusDormant;case"6","notpresent":return domain.LinkStatusNotPresent;case"7","lower_layer_down","downlowerlayer":return domain.LinkStatusLowerLayerDown};return domain.LinkStatusUnknown}
