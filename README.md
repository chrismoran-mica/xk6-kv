# xk6-kv

This is a [k6](https://go.k6.io/k6) extension using the [xk6](https://github.com/grafana/xk6) system.

| :exclamation: This is a proof of concept, isn't supported by the k6 team, and may break in the future. USE AT YOUR OWN RISK! |
|------|

### Note: I lifted this from the original xk6-kv [repo](https://github.com/dgzlopes/xk6-kv).

## Build

To build a `k6` binary with this extension, first ensure you have the prerequisites:

- [Go toolchain](https://go101.org/article/go-toolchain.html)
- Git

Then:

1. Install `xk6`:
  ```shell
  $ go install go.k6.io/xk6/cmd/xk6@latest
  ```

2. Build the binary:
  ```shell
  $ xk6 build v0.42.0 --with github.com/chrismoran-mica/xk6-kv@latest
  ```

## Example

```javascript
import {sleep} from 'k6'
import {Client} from 'k6/x/kv';

export const options = {
  scenarios: {
    generator: {
      exec: 'generator',
      executor: 'per-vu-iterations',
      vus: 5,
    },
    results: {
      exec: 'results',
      executor: 'per-vu-iterations',
      startTime: '1s',
      maxDuration: '2s',
      vus: 1,
    },
    ttl: {
      exec: 'ttl',
      executor: 'constant-vus',
      startTime: '3s',
      vus: 1,
      duration: '11s',
    },
  },
};

const client = new Client();

export function generator() {
  client.set(`hello_${__VU}`, 'world');
  client.setWithTTLInSecond(`ttl_${__VU}`, `ttl_${__VU}`, (+__VU + 3));
}

export function results() {
  console.log("Getting hello_1 and then deleting and getting again...");
  console.log(client.get("hello_1"));
  client.delete("hello_1");
  try {
    let keyDeleteValue = client.get("hello_1");
    console.log(typeof (keyDeleteValue));
  }
  catch (err) {
    console.log("empty value", err);
  }
  let r = client.viewPrefix("hello");
  for (let key in r) {
    console.log(key, r[key])
  }
}

export function ttl() {
  let r = client.viewPrefix("ttl");
  let count = 0;
  for (let key in r) {
    count++;
  }
  console.log(`count = ${count}`);
  sleep(1);
}

```

Result output:

```
$ ./k6 run example.js


          /\      |‾‾| /‾‾/   /‾‾/   
     /\  /  \     |  |/  /   /  /    
    /  \/    \    |     (   /   ‾‾\  
   /          \   |  |\  \ |  (‾)  | 
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: example.js
     output: -

  scenarios: (100.00%) 3 scenarios, 7 max VUs, 10m30s max duration (incl. graceful stop):
           * generator: 1 iterations for each of 5 VUs (maxDuration: 10m0s, exec: generator, gracefulStop: 30s)
           * results: 1 iterations for each of 1 VUs (maxDuration: 2s, exec: results, startTime: 1s, gracefulStop: 30s)
           * ttl: 1 looping VUs for 3s (exec: ttl, startTime: 3s, gracefulStop: 30s)

INFO[0001] Getting hello_1 and then deleting and getting again...  source=console
INFO[0001] world                                         source=console
INFO[0001] empty value {"value":{}}                      source=console10m0s  5/5 iters, 1 per VU
INFO[0001] hello_7 world                                 source=console      
INFO[0001] hello_2 world                                 source=console      
INFO[0001] hello_3 world                                 source=console
INFO[0001] hello_4 world                                 source=console
INFO[0001] hello_5 world                                 source=console
INFO[0001] hello_6 world                                 source=console
INFO[0003] empty value for 'ttl_0' {"value":{}}          source=console
INFO[0003] ttl_1                                         source=console
INFO[0003] ttl_2                                         source=console10m0s  5/5 iters, 1 per VU
INFO[0003] ttl_3                                         source=console       1/1 iters, 1 per VU
INFO[0003] empty value for 'ttl_4' {"value":{}}          source=console      
INFO[0004] empty value for 'ttl_0' {"value":{}}          source=console
INFO[0004] ttl_1                                         source=console
INFO[0004] ttl_2                                         source=consolem0s  5/5 iters, 1 per VU
INFO[0004] ttl_3                                         source=console     1/1 iters, 1 per VU
INFO[0004] empty value for 'ttl_4' {"value":{}}          source=console    
INFO[0005] empty value for 'ttl_0' {"value":{}}          source=console
INFO[0005] empty value for 'ttl_1' {"value":{}}          source=console
INFO[0005] empty value for 'ttl_2' {"value":{}}          source=consolem0s  5/5 iters, 1 per VU
INFO[0005] empty value for 'ttl_3' {"value":{}}          source=console     1/1 iters, 1 per VU
INFO[0005] empty value for 'ttl_4' {"value":{}}          source=console    

running (00m06.0s), 0/7 VUs, 9 complete and 0 interrupted iterations
generator ✓ [======================================] 5 VUs  00m00.0s/10m0s  5/5 iters, 1 per VU
results   ✓ [======================================] 1 VUs  0.0s/2s         1/1 iters, 1 per VU
ttl       ✓ [======================================] 1 VUs  3s             

     data_received........: 0 B 0 B/s
     data_sent............: 0 B 0 B/s
     iteration_duration...: avg=333.65ms min=60.48µs med=107.12µs max=1s p(90)=1s p(95)=1s
     iterations...........: 9   1.499347/s
     vus..................: 1   min=0      max=1
     vus_max..............: 7   min=7      max=7

```
