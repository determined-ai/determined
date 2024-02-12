import { ProjectSpec, StreamSpec } from "./projects";
import {forEach, reduce} from 'lodash'

// About 60 seconds of auto-retry.
const backoffs = [0, 1, 2, 4, 8, 10, 10, 10, 15]
const sleep = (s: number) => new Promise((r) => setTimeout(r, 1000 * s));

type Entity = {
    keyCache: KeyCache,
    spec: StreamSpec
}

class Stream implements Iterable<Record<string, any>> {

    #ws: WebSocket
    #retries: number = 0
    #num_syncs: number = 0

    #entities: Array<Record<string, Entity>> = []
    #cur_entity: Record<string, Entity> = {}

    #sync_sent: string | undefined = undefined
    #sync_complete: string | undefined = undefined

    // #caches: Record<string, KeyCache> = {}
    // #spec: Record<string, StreamSpec> = {}

    constructor(ws_url: string) {
        this.#ws = new WebSocket(ws_url)
        this.#ws.onopen = () => console.log("Streaming websocket opened!")
    }
    [Symbol.iterator](): Iterator<Record<string, any>, any, undefined> {
        throw new Error("Method not implemented.");
    }

    async _retry() {
        try {
            const backoff = backoffs[this.#retries]
            this.#retries += 1
            await sleep(backoff)
            return true
        } catch {
            return false
        }
    }

    send_spec(entity: Record<string, Entity>) {
        this.#num_syncs += 1
        const sync_id = this.#num_syncs.toString()
        const spec = reduce(entity, (spec, ent, k) => {
            spec["known"][k] = ent.keyCache.known() 
            spec["subscribe"][k] = {
                ...ent.spec.toWire(),
                "since": ent.keyCache.maxSeq()
            }
            return spec
        }, {"sync_id": sync_id, "known": {}, "subscribe": {}} as Record<string, any>)

        this.#ws.send(JSON.stringify(spec))
        this.#cur_entity = entity
        this.#sync_sent = sync_id
    }

    advance_subscription() {
        if(this.#ws.readyState !== 1) return
        if(!this.#cur_entity && this.#entities) return

        let entity: Record<string, Entity>
        if(!this.#sync_sent) {
            if(this.#cur_entity) {
                entity = this.#cur_entity
            } else {
                entity = this.#entities.shift() as Record<string, Entity>
            }
            this.send_spec(entity)
        } 
        if(this.#entities && this.#sync_complete === this.#sync_sent) {
            entity = this.#entities.shift() as Record<string, Entity>
            this.send_spec(entity)
        }

    }

    public subscribe(spec: StreamSpec) {
        const cur_spec = this.#cur_entity[spec.id()]
        let new_spec: Entity;
        if(cur_spec && !cur_spec.spec.equals(spec)) {
            new_spec = {spec, keyCache: cur_spec.keyCache.reset()}
        } else {
            new_spec = {spec, keyCache: new KeyCache()}
        }

        this.#entities.push({...this.#cur_entity, [spec.id()]: new_spec})
        this.advance_subscription()

        return this
    }
}