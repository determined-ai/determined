import { forEach, reduce, trimEnd } from 'lodash';

import { decode_keys, KeyCache } from './keyCache';
import { StreamSpec } from './projects';

import { Streamable, StreamEntityMap } from '.';

// About 60 seconds of auto-retry.
const backoffs = [0, 1, 2, 4, 8, 10, 10, 10, 15];
export const sleep = (s: number): Promise<unknown> => new Promise((r) => setTimeout(r, 1000 * s));

type Subscription = {
    keyCache: KeyCache,
    spec: StreamSpec
}

export class Stream {

    readonly #wsUrl: string;
    #ws: WebSocket | undefined = undefined;
    #retries: number = 0;
    #numSyncs: number = 0;
    #closedByClient: boolean = false;

    #subs: Array<Record<Streamable, Subscription>> = [];
    #curSub: Record<Streamable, Subscription> | undefined;

    // syncSent updates when msg sent to the stream
    #syncSent: string | undefined = undefined;
    // syncStarted updates when recieving msg of {sync_id, complated: false}
    #syncStarted: string | undefined = undefined;
    // syncStarted updates when recieving msg of {sync_id, complated: true}
    #syncComplete: string | undefined = undefined;

    //callbacks
    #onUpsert: (m: Record<string, any>) => void;
    #onDelete: (s: Streamable, a: Array<number>) => void;

    constructor(wsUrl: string, onUpsert: (m: Record<string, any>) => void, onDelete: (s: Streamable, a: Array<number>) => void) {
        this.#wsUrl = wsUrl;
        this.#onUpsert = onUpsert;
        this.#onDelete = onDelete;
        this.#connect();
    }

    #connect() {
        this.#ws = new WebSocket(this.#wsUrl);
        this.#ws.onopen = () => {
            console.log('Streaming websocket opened!');
            this.#advanceSubscription();
        };
        this.#ws.onerror = async (err) => {
            console.log('Streaming websocket errored: ', err);
            await this.#retry();
        };

        this.#ws.onclose = async () => {
            if (!this.#closedByClient) await this.#retry();

        };

        this.#ws.onmessage = (event) => {

            const msg = JSON.parse(event.data) as Record<string, any>;
            if (msg['sync_id']) {
                if (!msg['complete']) {
                    this.#syncStarted = msg['sync_id'];
                } else {
                    this.#syncComplete = msg['sync_id'];
                    this.#advanceSubscription();
                }
            } else if (this.#syncSent !== this.#syncStarted) {
                // Ignore all messages between when we send a new subscription and when the
                // sync-start message for that subscription arrives.  These are the online
                // updates for a subscription we no longer care about.
                return;
            } else {

                forEach(msg, (val, k) => {
                    if (k.includes('_deleted')) {
                        const stream_key = trimEnd(k, '_deleted') as Streamable;
                        const deleted_keys = decode_keys(val as string);
                        this.#curSub?.[stream_key].keyCache.delete_msg(deleted_keys);
                        this.#onDelete(stream_key, deleted_keys);
                    } else {
                        this.#curSub?.[StreamEntityMap[k]].keyCache.upsert([val.id], val.seq);
                        this.#onUpsert(msg);
                    }
                });

            }

        };
    }

    async #retry(): Promise<void> {
        this.#syncSent = undefined;
        const backoff = backoffs[this.#retries];
        this.#retries += 1;
        console.log(`#${this.#retries} of retries: in ${backoff}s`);
        await sleep(backoff);
        this.#connect();
    }

    #shouldSkip(newSub: Record<Streamable, Subscription>): boolean {
        if (!this.#curSub) return false;
        let same = true;
        forEach(newSub, (val, k) => {
            if (!this.#curSub?.[k as Streamable].spec.equals(val.spec)) same = false;
        });
        return same;
    }

    #sendSpec(newSub?: Record<Streamable, Subscription>): void {
        if (!newSub || !this.#ws) return;
        // Skip current sub and move to next only if sub has already been sent
        if (this.#shouldSkip(newSub) && this.#syncSent) {
            this.#advanceSubscription();
            return;
        }

        this.#numSyncs += 1;
        const sync_id = this.#numSyncs.toString();
        const spec = reduce(newSub, (spec, ent, k) => {
            spec['known'][k] = ent.keyCache.known();
            spec['subscribe'][k] = {
                ...ent.spec.toWire(),
                since: ent.keyCache.maxSeq(),
            };
            return spec;
        }, { known: {}, subscribe: {}, sync_id: sync_id } as Record<string, any>);

        this.#curSub = newSub;
        this.#ws.send(JSON.stringify(spec));
        this.#syncSent = sync_id;
    }

    #advanceSubscription(): void {
        if (!this.#ws || this.#ws.readyState !== this.#ws.OPEN) return;
        if (!this.#curSub && !this.#subs) return;

        let spec: Record<Streamable, Subscription> | undefined;
        // Just connected/reconnected
        if (!this.#syncSent) {
            // Resend current subscription if current sync not completed
            if (this.#curSub && (!this.#syncComplete || this.#syncStarted !== this.#syncComplete)) {
                spec = this.#curSub;
            } else {
                spec = this.#subs.shift();
            }

            this.#sendSpec(spec);
        } else if (this.#subs && this.#syncComplete === this.#syncSent) {
            // for established connection, only send a new sub when current sub is completed
            spec = this.#subs.shift();
            this.#sendSpec(spec);
        }
    }

    public subscribe(spec: StreamSpec): void {
        const curSpec = this.#curSub?.[spec.id()];
        const keyCache = curSpec && !curSpec.spec.equals(spec) ? curSpec.keyCache : new KeyCache();

        this.#subs.push({ ...this.#curSub, [spec.id()]: { keyCache, spec } });

        this.#advanceSubscription();
    }

    public close(): void {
        this.#closedByClient = true;
        this.#ws && this.#ws.close();
    }
}
