import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = 'http://localhost:8000';
const TOKEN = 'eyJzdWIiOiI5ZThlNjVlZS0xNzU5LTQ1YWQtOTE4NS1iMGJiYWU2ZDJjZWIiLCJyb2xlIjoiYWRtaW4iLCJpYXQiOjE3Nzg4NTI4MzIsImV4cCI6MTc3ODg4MTYzMn0.ruyJVFdrkQHtWO1bwYVr-FPeofAxcveX7StUccTe5hs';

const headers = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${TOKEN}`,
};

export const options = {
  stages: [
    { duration: '30s', target: 50 },
    { duration: '1m',  target: 500 },
    { duration: '1m',  target: 2000 },
    { duration: '1m',  target: 5000 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed:   ['rate<0.05'],
  },
};

const TOWER_IDS = [
  '00000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000003',
  '0bf4f03a-b52a-4300-b8a8-d18baa5a1432',
  '1a01d16f-ca8c-4dd7-87c7-0bf50d952a28',
];

export default function () {
  const towerId = TOWER_IDS[Math.floor(Math.random() * TOWER_IDS.length)];

  // teste 1 - listar torres
  const r1 = http.get(`${BASE_URL}/api/v1/towers`, { headers });
  check(r1, { 'towers 200': (r) => r.status === 200 });

  // teste 2 - detalhe de torre
  const r2 = http.get(`${BASE_URL}/api/v1/towers/${towerId}`, { headers });
  check(r2, { 'tower detail 200': (r) => r.status === 200 });

  // teste 3 - ingestão de métrica
  const payload = JSON.stringify({
    tower_id: towerId,
    collected_at: new Date().toISOString(),
    metrics: {
      signal_strength: Math.floor(Math.random() * 40) - 100,
      voltage: 48 + Math.random() * 10,
    },
  });
  const r3 = http.post(`${BASE_URL}/api/v1/metrics`, payload, { headers });
  check(r3, { 'metrics 202': (r) => r.status === 202 });

  sleep(0.1);
}
