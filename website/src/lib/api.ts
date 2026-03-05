export const API = 'http://localhost:8080';
export const PAGE_SIZE = 20;

export interface KVPair {
    key: string;
    value: string;
}

export interface PageResult {
    pairs: KVPair[];
    total: number;
}

export async function fetchPage(page: number, pageSize: number = PAGE_SIZE): Promise<PageResult> {
    const offset = (page - 1) * pageSize;
    const res = await fetch(`${API}/keys?offset=${offset}&limit=${pageSize}`);
    if (!res.ok) throw new Error(`Failed to fetch keys: ${res.status}`);
    const data: { keys: string[]; total: number } = await res.json();

    if (data.keys.length === 0) return { pairs: [], total: data.total };

    const keysList = data.keys.map(k => `"${k.replace(/"/g, '\\"')}"`).join(', ');
    const query = `REVEAL (${keysList})`;
    const kvRes = await runQuery(query);

    let pairs: KVPair[] = [];
    if (kvRes.result && Array.isArray(kvRes.result)) {
        pairs = kvRes.result.map((row: any) => ({
            key: row.key,
            value: row.found !== false ? (row.value ?? '') : ''
        }));
    } else if (kvRes.result && typeof kvRes.result === 'object') {
        const row: any = kvRes.result;
        pairs = [{
            key: row.key,
            value: row.found !== false ? (row.value ?? '') : ''
        }];
    } else {
        pairs = data.keys.map(key => ({ key, value: '' }));
    }

    return { pairs, total: data.total };
}

export async function runQuery(query: string): Promise<{ result?: unknown; error?: string }> {
    const res = await fetch(`${API}/query`, { method: 'POST', body: query });
    const data = await res.json();
    if (!res.ok || data.error) return { error: data.error ?? `HTTP ${res.status}` };
    return { result: data.result };
}
