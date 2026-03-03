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

    const pairs = await Promise.all(
        data.keys.map(async (key) => {
            try {
                const valRes = await fetch(`${API}/get?key=${encodeURIComponent(key)}`);
                if (!valRes.ok) return { key, value: '' };
                const val = await valRes.json();
                return { key, value: val.value ?? '' };
            } catch {
                return { key, value: '' };
            }
        })
    );

    return { pairs, total: data.total };
}

export async function runQuery(query: string): Promise<{ result?: unknown; error?: string }> {
    const res = await fetch(`${API}/query`, { method: 'POST', body: query });
    const data = await res.json();
    if (!res.ok || data.error) return { error: data.error ?? `HTTP ${res.status}` };
    return { result: data.result };
}
