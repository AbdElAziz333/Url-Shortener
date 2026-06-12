import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const createDuration = new Trend('create_link_duration', true);
const listDuration   = new Trend('list_links_duration', true);

// ── Tune these ────────────────────────────────────────────────────────────────
const BASE_URL  = 'http://localhost:8080';
const TEST_USER = { email: 'perf-shortener@test.com', password: 'PerfTest123!' };
// ─────────────────────────────────────────────────────────────────────────────

export const options = {
  stages: [
    { duration: '15s', target: 10 },
    { duration: '1m',  target: 10 },
    { duration: '15s', target: 30 },
    { duration: '30s', target: 30 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed:     ['rate<0.05'],
    create_link_duration:['p(95)<2000'],
    list_links_duration: ['p(95)<1000']
  },
};

export function setup() {
  http.post(
    `${BASE_URL}/gateway/auth/register`,
    JSON.stringify(TEST_USER),
    { headers: { 'Content-Type': 'application/json' } }
  );

  const loginRes = http.post(
    `${BASE_URL}/gateway/auth/login`,
    JSON.stringify(TEST_USER),
    { headers: { 'Content-Type': 'application/json' } }
  );

  const token = loginRes.headers['Access_token'];
  if (!token) {
    console.error(`Login failed! Status: ${loginRes.status}, Body: ${loginRes.body}`);
    return { token: null };
  }

  console.log(`Login successful. Token: ${token.substring(0, 20)}...`);
  return { token };
}

const SAMPLE_URLS = [
  'https://github.com',
  'https://google.com',
  'https://stackoverflow.com',
  'https://youtube.com',
  'https://reddit.com',
  'https://twitter.com',
  'https://linkedin.com',
  'https://medium.com',
];

export default function (data) {
  if (!data.token) {
    console.error('No token available, skipping VU');
    return;
  }

  const authHeaders = {
    headers: {
      'Authorization': `Bearer ${data.token}`,
      'Content-Type': 'application/json',
    },
  };

  const randomURL = SAMPLE_URLS[Math.floor(Math.random() * SAMPLE_URLS.length)];
  const createStart = Date.now();
  const createRes = http.post(
    `${BASE_URL}/shortener/api/links`,
    JSON.stringify({ original_url: randomURL }),
    authHeaders,
  );
  
  createDuration.add(Date.now() - createStart);

  check(createRes, {
    'create link status 200': (r) => r.status === 200,
    'create link has data':   (r) => r.json('data') !== null,
  });

  errorRate.add(createRes.status !== 200);

  // sleep(0.5);
  
  const listStart = Date.now();
  const listRes = http.get(`${BASE_URL}/shortener/api/links`, authHeaders);
  listDuration.add(Date.now() - listStart);

  check(listRes, {
    'list links status 200': (r) => r.status === 200,
    'list links has data':   (r) => r.json('data') !== null,
  });

  errorRate.add(listRes.status !== 200);

  // sleep(1);
}
