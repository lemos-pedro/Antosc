// ============================================================
// AIP — Snippets Power Query (M) para Power BI Desktop
// Substituir HOST e API_KEY
// ============================================================

// --- Parâmetro recomendado (criar em Gerir parâmetros) ---
// apiKey  = "seu-aip-api-key"
// host    = "http://localhost:8090"
// pFrom   = "2026-01-01"
// pTo     = "2026-12-31"

// ---------- towers ----------
let
    Fonte = Json.Document(
        Web.Contents(
            "http://HOST:8090/api/v1/powerbi/towers",
            [Headers=[#"X-API-Key"="API_KEY"]]
        )
    ),
    Tabela = Table.FromList(Fonte, Splitter.SplitByNothing(), null, null, ExtraValues.Error),
    Expand = Table.ExpandRecordColumn(Tabela, "Column1",
        {"tower_id","name","vendor","operator_id","region_id","availability_7d","availability_30d"},
        {"tower_id","name","vendor","operator_id","region_id","availability_7d","availability_30d"}),
    Tipos = Table.TransformColumnTypes(Expand, {
        {"availability_7d", type number},
        {"availability_30d", type number}
    })
in
    Tipos

// ---------- incidents ----------
let
    Url = "http://HOST:8090/api/v1/powerbi/incidents?from=2026-01-01&to=2026-12-31",
    Fonte = Json.Document(Web.Contents(Url, [Headers=[#"X-API-Key"="API_KEY"]])),
    Tabela = Table.FromList(Fonte, Splitter.SplitByNothing(), null, null, ExtraValues.Error),
    Expand = Table.ExpandRecordColumn(Tabela, "Column1",
        {"id","tower_id","incident_started_at","incident_ended_at","ml_predicted_cause",
         "ml_confidence","confirmed_cause","confirmed_by","confirmed_at","status","created_at"}),
    Tipos = Table.TransformColumnTypes(Expand, {
        {"incident_started_at", type datetime},
        {"incident_ended_at", type datetime},
        {"confirmed_at", type datetime},
        {"created_at", type datetime},
        {"ml_confidence", type number}
    })
in
    Tipos

// ---------- consumption ----------
let
    Url = "http://HOST:8090/api/v1/powerbi/consumption?from=2026-01-01&to=2026-12-31",
    Fonte = Json.Document(Web.Contents(Url, [Headers=[#"X-API-Key"="API_KEY"]])),
    Tabela = Table.FromList(Fonte, Splitter.SplitByNothing(), null, null, ExtraValues.Error),
    Expand = Table.ExpandRecordColumn(Tabela, "Column1",
        {"tower_id","equipment_type","expected_value","average_actual","unit",
         "tolerance_percent","deviation_percent","within_norm","sample_count","period_from","period_to"}),
    Tipos = Table.TransformColumnTypes(Expand, {
        {"expected_value", type number},
        {"average_actual", type number},
        {"tolerance_percent", type number},
        {"deviation_percent", type number},
        {"within_norm", type logical},
        {"sample_count", Int64.Type}
    })
in
    Tipos

// ---------- predictions ----------
let
    Fonte = Json.Document(
        Web.Contents(
            "http://HOST:8090/api/v1/powerbi/predictions",
            [Headers=[#"X-API-Key"="API_KEY"]]
        )
    ),
    Tabela = Table.FromList(Fonte, Splitter.SplitByNothing(), null, null, ExtraValues.Error),
    Expand = Table.ExpandRecordColumn(Tabela, "Column1",
        {"tower_id","model","score","status","explanation","prediction_window_days","created_at"}),
    Tipos = Table.TransformColumnTypes(Expand, {
        {"score", type number},
        {"prediction_window_days", Int64.Type},
        {"created_at", type datetime}
    })
in
    Tipos

// ---------- kpis ----------
let
    Url = "http://HOST:8090/api/v1/powerbi/kpis?from=2026-01-01&to=2026-12-31",
    Fonte = Json.Document(Web.Contents(Url, [Headers=[#"X-API-Key"="API_KEY"]])),
    Tabela = Table.FromRecords({Fonte})
in
    Tabela
