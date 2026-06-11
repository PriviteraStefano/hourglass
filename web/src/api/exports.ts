const API_BASE = '/api'

export function getExportUrl(type: 'timesheets' | 'expenses' | 'combined', from: string, to: string): string {
  const params = new URLSearchParams({ from, to })
  return `${API_BASE}/exports/${type}?${params}`
}
