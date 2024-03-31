import http from "k6/http";
import { check } from "k6";

// k6 run --vus 10 --duration 10s k6/list.js
export default function () {
    const url = 'http://127.0.0.1:3000/v1/playlist';

    let res = http.get(url);

    check(res, {
        "status was 200": (r) => r.status == 200,
        'verify result': (r) =>
            r.body.includes('1190bf92-10dc-4ce5-968a-7a377f37f984'),
    });
}
