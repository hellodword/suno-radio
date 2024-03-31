import http from "k6/http";
import { check } from "k6";

// k6 run --vus 10 --duration 120s k6/radio.js 2>/dev/null
export default function () {
    const url = 'http://127.0.0.1:3000/v1/playlist/1190bf92-10dc-4ce5-968a-7a377f37f984';

    const param = { timeout: "10s" };

    let res = http.get(url, param);

    check(res, {
        'verify mp3': (r) => r.body.startsWith('ID3'),
        'verify result': (r) => r.body.length > 4096,
    });
}