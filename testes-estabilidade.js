import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = 'http://localhost:8000';
const TEST_TOWERS = 5;

export const options = {
  stages: [
    { duration: '30s', target: 50 },
    { duration: '1m', target: 100 },
    { duration: '1m', target: 200 },
    { duration: '1m', target: 500 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.05'],
  },
};

export function setup() {
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ username: 'admin', password: 'admin123' }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(loginRes, { 'login 200': (r) => r.status === 200 });
  const token = loginRes.json('access_token');
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  };

  const towersRes = http.get(`${BASE_URL}/api/v1/towers`, { headers });
  check(towersRes, { 'list towers 200': (r) => r.status === 200 });

  let towers = towersRes.json();
  if (!Array.isArray(towers)) {
    towers = [];
  }

  while (towers.length < TEST_TOWERS) {
    const idx = towers.length + 1;
    const createRes = http.post(
      `${BASE_URL}/api/v1/towers`,
      JSON.stringify({
        name: `LoadTest-Tower-${idx}`,
        status: 'online',
        vendor: 'eltek',
        snmp_enabled: false,
        availability_30d: 99.9,
      }),
      { headers }
    );

    check(createRes, { 'create tower 201': (r) => r.status === 201 });
    if (createRes.status >= 200 && createRes.status < 300) {
      towers.push(createRes.json());
    } else {
      break;
    }
  }

  const towerIDs = towers
    .map((tower) => tower?.tower_id)
    .filter((towerID) => typeof towerID === 'string' && towerID.length > 0)
    .slice(0, TEST_TOWERS);

  if (towerIDs.length === 0) {
    throw new Error('No valid tower IDs available for the load test');
  }

  return { token, towerIDs };
}

export default function (data) {
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${data.token}`,
  };

  const towerId = data.towerIDs[Math.floor(Math.random() * data.towerIDs.length)];

  const r1 = http.get(`${BASE_URL}/api/v1/towers`, { headers });
  check(r1, { 'towers 200': (r) => r.status === 200 });

  const r2 = http.get(`${BASE_URL}/api/v1/towers/${towerId}`, { headers });
  check(r2, { 'tower detail 200': (r) => r.status === 200 });

  const payload = JSON.stringify({
    tower_id: towerId,
    collected_at: new Date().toISOString(),
    metrics: {
      signal_strength: Math.floor(Math.random() * 40) - 100,
      voltage: 48 + Math.random() * 10,
    },
  });
  const r3 = http.post(`${BASE_URL}/api/v1/metrics`, payload, { headers });
  check(r3, { 'metrics 2xx': (r) => r.status >= 200 && r.status < 300 });

  sleep(0.1);
}
//k6 run ~/Documentos/Antosc-system/testes-estabilidade.js