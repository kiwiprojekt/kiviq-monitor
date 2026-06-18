export function computeRates(raw, field) {
  if (raw.length < 2) return []
  const result = []
  for (let i = 1; i < raw.length; i++) {
    const dt = (raw[i].t - raw[i - 1].t) / 1000
    if (dt <= 0) continue
    const dv = (raw[i][field] || 0) - (raw[i - 1][field] || 0)
    result.push({ t: raw[i].t, val: Math.max(0, dv / dt) })
  }
  return result
}

// normalizeHistoryPoint turns a live WS update into a chart point. The agent's
// HistoryPoint carries its timestamp in Unix SECONDS (same as the /history
// endpoint), but the charts plot against Date.now() in milliseconds, so the
// timestamp must be scaled. When no history point is present (older snapshots),
// fall back to the current snapshot's CPU/memory at the current time.
export function normalizeHistoryPoint(snap, historyPoint) {
  if (historyPoint) {
    return { ...historyPoint, t: historyPoint.t * 1000 }
  }
  return {
    t: Date.now(),
    cpu: snap?.cpu?.usage_percent || 0,
    mem: snap?.memory?.usage_percent || 0,
    rx: 0, tx: 0, dr: 0, dw: 0,
  }
}

// mergeLivePoint appends a live point to a series in place, ignoring points that
// are not strictly newer than the last (out-of-order or duplicate WS updates)
// and dropping points older than cutoff. At least one point is always kept.
export function mergeLivePoint(points, pt, cutoff) {
  const last = points[points.length - 1]
  if (last && pt.t <= last.t) return points
  points.push(pt)
  while (points.length > 1 && points[0].t < cutoff) points.shift()
  return points
}

// sampleSeriesAt evaluates a series at the monotonically increasing times
// t = tStart + i*tStep for i in [0, count]. It is equivalent to calling
// interpolateAt at each time, but runs in a single O(count + data.length)
// forward walk instead of O(count * data.length): because both the sample
// times and data points are sorted, the bracketing cursor only ever advances.
// Returns an array of length count+1 (all null when data is empty).
export function sampleSeriesAt(data, tStart, tStep, count) {
  const out = new Array(count + 1)
  if (data.length === 0) return out.fill(null)

  const last = data[data.length - 1]
  let j = 0 // data[j] is the lower bound of the bracket for the current time
  for (let i = 0; i <= count; i++) {
    const t = tStart + i * tStep
    if (t <= data[0].t) { out[i] = data[0].val; continue }
    if (t >= last.t) { out[i] = last.val; continue }
    while (j < data.length - 1 && data[j + 1].t <= t) j++
    const prev = data[j]
    const next = data[j + 1]
    const dt = next.t - prev.t
    out[i] = dt <= 0 ? next.val : prev.val + (next.val - prev.val) * ((t - prev.t) / dt)
  }
  return out
}

export function interpolateAt(data, t) {
  if (data.length === 0) return null
  if (data.length === 1) return data[0].val
  if (t <= data[0].t) return data[0].val
  if (t >= data[data.length - 1].t) return data[data.length - 1].val
  for (let i = 1; i < data.length; i++) {
    if (data[i].t >= t) {
      const prev = data[i - 1]
      const next = data[i]
      const dt = next.t - prev.t
      if (dt <= 0) return next.val
      const frac = (t - prev.t) / dt
      return prev.val + (next.val - prev.val) * frac
    }
  }
  return data[data.length - 1].val
}
