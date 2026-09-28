import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const redirectDuration = new Trend('redirect_duration', true);

const BASE_URL = 'http://localhost:8080';
const TEST_USER = { email: 'perf-redirect@test.com', password: 'PerfTest123!' };

// ── Tune these for fair REST vs gRPC comparison ─────────────────────────────
// Target requests per second (not VUs). Same rate = same load regardless of
// how fast each transport completes an iteration.
const WARMUP_RPS = 50;
const PEAK_RPS = 200;
// ─────────────────────────────────────────────────────────────────────────────

export const options = {
  scenarios: {
    redirect_ramp: {
      executor: 'ramping-arrival-rate',
      startRate: WARMUP_RPS,
      timeUnit: '1s',
      preAllocatedVUs: WARMUP_RPS,
      maxVUs: PEAK_RPS * 3,
      stages: [
        { target: WARMUP_RPS, duration: '15s' },
        { target: WARMUP_RPS, duration: '1m' },
        { target: PEAK_RPS, duration: '15s' },
        { target: PEAK_RPS, duration: '30s' },
        { target: 0, duration: '10s' },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    redirect_duration: ['p(95)<200', 'p(99)<500'],
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
    return { code: null };
  }

  const createRes = http.post(
    `${BASE_URL}/shortener/api/links`,
    JSON.stringify({ original_url: 'https://example.com' }),
    {
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    }
  );

  const data = createRes.json('data');
  if (!data || !data.code) {
    console.error(`Failed to create link! Status: ${createRes.status}, Body: ${createRes.body}`);
    return { code: null };
  }

  console.log(`Test link created. Code: ${data.code}`);
  return { code: data.code };
}

export default function (data) {
  if (!data.code) {
    console.error('No short code available, skipping iteration');
    return;
  }

  const start = Date.now();
  const res = http.get(`${BASE_URL}/redirect/${data.code}`, { redirects: 0 });
  redirectDuration.add(Date.now() - start);

  const ok = check(res, {
    'redirect status 302': (r) => r.status === 302,
    'location header set': (r) => r.headers.Location !== undefined,
  });

  errorRate.add(!ok);
}
