import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const createDuration = new Trend('create_link_duration', true);
const listDuration = new Trend('list_links_duration', true);

const BASE_URL = 'http://localhost:8080';
const TEST_USER = { email: 'perf-shortener@test.com', password: 'PerfTest123!' };

// ── Tune these for fair REST vs gRPC comparison ─────────────────────────────
// Rate = iterations/s. Each iteration = 1 create + 1 list (2 HTTP requests).
// Peak ~90 iter/s ≈ 180 HTTP req/s, matching the old 30-VU sustained load.
const WARMUP_ITER_PER_S = 30;
const PEAK_ITER_PER_S = 90;
// ─────────────────────────────────────────────────────────────────────────────

export const options = {
  scenarios: {
    shortener_ramp: {
      executor: 'ramping-arrival-rate',
      startRate: WARMUP_ITER_PER_S,
      timeUnit: '1s',
      preAllocatedVUs: WARMUP_ITER_PER_S * 2,
      maxVUs: PEAK_ITER_PER_S * 3,
      stages: [
        { target: WARMUP_ITER_PER_S, duration: '15s' },
        { target: WARMUP_ITER_PER_S, duration: '1m' },
        { target: PEAK_ITER_PER_S, duration: '15s' },
        { target: PEAK_ITER_PER_S, duration: '30s' },
        { target: 0, duration: '10s' },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
    create_link_duration: ['p(95)<2000'],
    list_links_duration: ['p(95)<1000'],
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
    console.error('No token available, skipping iteration');
    return;
  }

  const authHeaders = {
    headers: {
      Authorization: `Bearer ${data.token}`,
      'Content-Type': 'application/json',
    },
  };

  const randomURL = SAMPLE_URLS[Math.floor(Math.random() * SAMPLE_URLS.length)];

  const createStart = Date.now();
  const createRes = http.post(
    `${BASE_URL}/shortener/api/links`,
    JSON.stringify({ original_url: randomURL }),
    authHeaders
  );
  createDuration.add(Date.now() - createStart);

  check(createRes, {
    'create link status 200': (r) => r.status === 200,
    'create link has data': (r) => r.json('data') !== null,
  });
  errorRate.add(createRes.status !== 200);

  const listStart = Date.now();
  const listRes = http.get(`${BASE_URL}/shortener/api/links`, authHeaders);
  listDuration.add(Date.now() - listStart);

  check(listRes, {
    'list links status 200': (r) => r.status === 200,
    'list links has data': (r) => r.json('data') !== null,
  });
  errorRate.add(listRes.status !== 200);
}
