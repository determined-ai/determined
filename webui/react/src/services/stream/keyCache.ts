
function* decode_keys(keys: string): Iterable<number>{
    if (keys.length === 0) {
        return 
    }
    for(const key of keys.split(",")) {
        if(key.includes("-")) {
            const [start, end] = key.split("-")
            for(let i = Number(start); i <= Number(end); i++) {
                yield i
            }
        } else {
            yield Number(key)
        }
    }
}

function encode_keys(set: Set<number>): string {
    if(!set) return ""
    const out: Array<string> = []
    const keys = Array.from(set).sort().values()
    let start: number = keys.next().value
    let end = start

    const emit = (start: number, end: number, out: Array<string>) => {
        if(start === end) {
            out.push(start.toString())
        } else {
            out.push(`${start}-${end}`)
        }
    }

    for (let k of keys) {
        if(k === end + 1) {
            end = k
            continue
        }
        // end of a range
        emit(start, end, out)
        start = k
        end = start
    }
    emit(start, end, out)

    return out.join(",")
}

class KeyCache {

    #keys: Set<number>
    #maxseq: number

    constructor(keys?: Set<number>) {
        this.#keys = keys || new Set()
        this.#maxseq = 0
    }

    delete_one(id: number) {
        this.#keys.delete(id)
    }

    public upsert(id: number, seq: number) {
        this.#keys.add(id)
        this.#maxseq = Math.max(this.#maxseq, seq)
    } 

    public delete_msg(deleted: string) {
        for(let id of decode_keys(deleted)) {
            this.delete_one(id)
        }
    }

    public known() {
        return encode_keys(this.#keys)
    }

    public maxSeq() {
        return this.#maxseq
    }

    public reset() {
        return new KeyCache(this.#keys)
    }
}