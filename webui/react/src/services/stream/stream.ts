import { forEach, reduce, trimEnd } from 'lodash';

import { decode_keys, KeyCache } from './keyCache';
import { StreamSpec } from './projects';

import { Streamable, StreamEntityMap } from '.';

// About 60 seconds of auto-retry.
const backoffs = [0, 1, 2, 4, 8, 10, 10, 10, 15];
const sleep = (s: number) => new Promise((r) => setTimeout(r, 1000 * s));

type Entity = {
    keyCache: KeyCache,
    spec: StreamSpec
}

export class Stream {

    #ws: WebSocket;
    #retries: number = 0;
    #numSyncs: number = 0;

    #entities: Array<Record<Streamable, Entity>> = [];
    #curEntity: Record<Streamable, Entity> | undefined;

    #syncSent: string | undefined = undefined;
    #syncComplete: string | undefined = undefined;
    #syncStarted: string |undefined = undefined;

    constructor(wsUrl: string, onUpsert: (m: Record<string, any>) => void, onDelete: (s: Streamable, a: Array<number>) => void) {
        this.#ws = new WebSocket(wsUrl);
        this.#ws.onopen = () => {
            console.log('Streaming websocket opened!');
            this.advanceSubscription();
        };
        this.#ws.onerror = async (err) => {
            console.log('Streaming websocket errored: ', err);
            console.log('# of retries: ', this.#retries);
            const wait = await this.retry();
            if (!wait) throw err;
            this.#ws.dispatchEvent(new Event('open'));
        };

        this.#ws.onmessage = (event) => {

            const msg = JSON.parse(event.data) as Record<string, any>;
            if (msg['sync_id']) {
                if (!msg['complete']) {
                    this.#syncStarted = msg['sync_id'];
                } else {
                    this.#syncComplete = msg['sync_id'];
                    this.advanceSubscription();
                }
            } else {
                // Ignore all messages between when we send a new subscription and when the
                // sync-start message for that subscription arrives.  These are the online
                // updates for a subscription we no longer care about.
                if (this.#syncSent !== this.#syncStarted) return;

                forEach(msg, (val, k) => {
                    if (k.includes('_deleted')) {
                        const stream_key = trimEnd(k, '_deleted') as Streamable;
                        const deleted_keys = decode_keys(val as string);
                        this.#curEntity?.[stream_key].keyCache.delete_msg(deleted_keys);
                        onDelete(stream_key, deleted_keys);
                    } else {
                        this.#curEntity?.[StreamEntityMap[k]].keyCache.upsert([val.id], val.seq);
                        onUpsert(msg);
                    }
                });

            }

        };
    }

    async retry(): Promise<boolean> {
        try {
            const backoff = backoffs[this.#retries];
            this.#retries += 1;
            await sleep(backoff);
            return true;
        } catch {
            return false;
        }
    }

    #sameAsCur(entity: Record<Streamable, Entity>): boolean {
        if (!this.#curEntity) return false;
        let same = true;
        forEach(entity, (val, k) => {
            if (!this.#curEntity?.[k as Streamable].spec.equals(val.spec)) same = false;
        });
        return same;
    }

    #sendSpec(entity?: Record<Streamable, Entity>): void {
        if (!entity || this.#sameAsCur(entity)) return;

        this.#numSyncs += 1;
        const sync_id = this.#numSyncs.toString();
        const spec = reduce(entity, (spec, ent, k) => {
            spec['known'][k] = ent.keyCache.known();
            spec['subscribe'][k] = {
                ...ent.spec.toWire(),
                since: ent.keyCache.maxSeq(),
            };
            return spec;
        }, { known: {}, subscribe: {}, sync_id: sync_id } as Record<string, any>);

        this.#ws.send(JSON.stringify(spec));
        this.#curEntity = entity;
        this.#syncSent = sync_id;
    }

    advanceSubscription(): void {
        if (this.#ws.readyState !== this.#ws.OPEN) return;
        if (!this.#curEntity && !this.#entities) return;

        let spec: Record<Streamable, Entity> | undefined;
        if (!this.#syncSent) {
            if (this.#curEntity) {
                spec = this.#curEntity;
            } else {
                spec = this.#entities.shift();
            }
            this.#sendSpec(spec);
        }
        if (this.#entities && this.#syncComplete === this.#syncSent) {
            spec = this.#entities.shift();
            this.#sendSpec(spec);
        }
    }

    public subscribe(spec: StreamSpec, known?: Array<number>): Stream {
        const curSpec = this.#curEntity?.[spec.id()];
        const keyCache = curSpec && !curSpec.spec.equals(spec) ? curSpec.keyCache : new KeyCache();
        if (known) {
            keyCache.upsert(known);
        }

        this.#entities.push({ ...this.#curEntity, [spec.id()]: { keyCache, spec } });
        this.advanceSubscription();

        return this;
    }

    public close(): void {
        this.#ws.close();
    }
}
