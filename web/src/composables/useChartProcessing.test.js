import { describe, it, expect, vi } from 'vitest'
import { computeRates, interpolateAt, sampleSeriesAt, normalizeHistoryPoint, mergeLivePoint } from './useChartProcessing.js'

describe('sampleSeriesAt', () => {
  it('returns nulls for an empty series', () => {
    expect(sampleSeriesAt([], 0, 10, 3)).toEqual([null, null, null, null])
  })

  it('matches interpolateAt at every sampled time (single forward walk)', () => {
    const data = [
      { t: 1000, val: 10 },
      { t: 2000, val: 30 },
      { t: 2500, val: 25 },
      { t: 4000, val: 80 },
      { t: 7000, val: 5 },
    ]
    // Sweep well before the first point to well after the last, so the clamped
    // ends and the interior interpolation are all exercised.
    const tStart = 0
    const tStep = 37 // deliberately not aligned to any point boundary
    const count = 250
    const got = sampleSeriesAt(data, tStart, tStep, count)
    expect(got).toHaveLength(count + 1)
    for (let i = 0; i <= count; i++) {
      const t = tStart + i * tStep
      expect(got[i]).toBeCloseTo(interpolateAt(data, t), 10)
    }
  })

  it('handles a single-point series (flat)', () => {
    expect(sampleSeriesAt([{ t: 500, val: 7 }], 0, 100, 3)).toEqual([7, 7, 7, 7])
  })
})

describe('computeRates', () => {
  it('returns empty for fewer than 2 points', () => {
    expect(computeRates([], 'cpu')).toEqual([])
    expect(computeRates([{ t: 1000, cpu: 50 }], 'cpu')).toEqual([])
  })

  it('computes rate between two points', () => {
    const raw = [
      { t: 1000, cpu: 0 },
      { t: 2000, cpu: 100 },
    ]
    const result = computeRates(raw, 'cpu')
    expect(result).toHaveLength(1)
    expect(result[0].val).toBeCloseTo(100)
    expect(result[0].t).toBe(2000)
  })

  it('clamps negative rates to zero', () => {
    const raw = [
      { t: 1000, cpu: 100 },
      { t: 2000, cpu: 0 },
    ]
    const result = computeRates(raw, 'cpu')
    expect(result[0].val).toBe(0)
  })

  it('skips zero-dt points', () => {
    const raw = [
      { t: 1000, cpu: 10 },
      { t: 1000, cpu: 20 },
      { t: 2000, cpu: 30 },
    ]
    const result = computeRates(raw, 'cpu')
    expect(result).toHaveLength(1)
    expect(result[0].t).toBe(2000)
  })

  it('handles multiple intervals', () => {
    const raw = [
      { t: 0, rx: 0 },
      { t: 1000, rx: 1024 },
      { t: 2000, rx: 2048 },
    ]
    const result = computeRates(raw, 'rx')
    expect(result).toHaveLength(2)
    expect(result[0].val).toBeCloseTo(1024)
    expect(result[1].val).toBeCloseTo(1024)
  })

  it('handles missing field values', () => {
    const raw = [
      { t: 0 },
      { t: 1000, cpu: 50 },
    ]
    const result = computeRates(raw, 'cpu')
    expect(result[0].val).toBeCloseTo(50)
  })
})

describe('interpolateAt', () => {
  it('returns null for empty data', () => {
    expect(interpolateAt([], 1000)).toBeNull()
  })

  it('returns the only value for single-point data', () => {
    expect(interpolateAt([{ t: 1000, val: 42 }], 500)).toBe(42)
  })

  it('clamps to first value when t is before range', () => {
    const data = [{ t: 1000, val: 10 }, { t: 2000, val: 20 }]
    expect(interpolateAt(data, 500)).toBe(10)
  })

  it('clamps to last value when t is after range', () => {
    const data = [{ t: 1000, val: 10 }, { t: 2000, val: 20 }]
    expect(interpolateAt(data, 3000)).toBe(20)
  })

  it('interpolates linearly between points', () => {
    const data = [{ t: 0, val: 0 }, { t: 1000, val: 100 }]
    expect(interpolateAt(data, 500)).toBeCloseTo(50)
    expect(interpolateAt(data, 250)).toBeCloseTo(25)
    expect(interpolateAt(data, 750)).toBeCloseTo(75)
  })

  it('returns exact values at data points', () => {
    const data = [{ t: 0, val: 10 }, { t: 1000, val: 20 }]
    expect(interpolateAt(data, 0)).toBe(10)
    expect(interpolateAt(data, 1000)).toBe(20)
  })

  it('handles zero-dt gracefully', () => {
    const data = [{ t: 1000, val: 10 }, { t: 1000, val: 20 }]
    expect(interpolateAt(data, 1000)).toBe(10)
  })
})

describe('normalizeHistoryPoint', () => {
  it('converts the agent timestamp from seconds to milliseconds', () => {
    const pt = normalizeHistoryPoint({}, { t: 1_700_000_000, cpu: 50, mem: 30 })
    expect(pt.t).toBe(1_700_000_000_000)
  })

  it('preserves the metric fields of a agent point', () => {
    const pt = normalizeHistoryPoint({}, { t: 1000, cpu: 12, mem: 34, rx: 5, tx: 6, dr: 7, dw: 8 })
    expect(pt).toMatchObject({ cpu: 12, mem: 34, rx: 5, tx: 6, dr: 7, dw: 8 })
  })

  it('falls back to the snapshot when no history point is supplied', () => {
    vi.spyOn(Date, 'now').mockReturnValue(2_000_000)
    const snap = { cpu: { usage_percent: 77 }, memory: { usage_percent: 55 } }
    const pt = normalizeHistoryPoint(snap, null)
    expect(pt).toEqual({ t: 2_000_000, cpu: 77, mem: 55, rx: 0, tx: 0, dr: 0, dw: 0 })
    Date.now.mockRestore()
  })

  it('defaults missing snapshot metrics to zero', () => {
    const pt = normalizeHistoryPoint({}, null)
    expect(pt.cpu).toBe(0)
    expect(pt.mem).toBe(0)
  })
})

describe('mergeLivePoint', () => {
  it('appends a newer point', () => {
    const points = [{ t: 1000, val: 1 }]
    mergeLivePoint(points, { t: 2000, val: 2 }, 0)
    expect(points).toHaveLength(2)
    expect(points[1]).toEqual({ t: 2000, val: 2 })
  })

  it('appends to an empty series', () => {
    const points = []
    mergeLivePoint(points, { t: 2000, val: 2 }, 0)
    expect(points).toEqual([{ t: 2000, val: 2 }])
  })

  it('ignores a point that is not newer than the last', () => {
    const points = [{ t: 2000, val: 2 }]
    mergeLivePoint(points, { t: 2000, val: 9 }, 0)
    mergeLivePoint(points, { t: 1500, val: 9 }, 0)
    expect(points).toEqual([{ t: 2000, val: 2 }])
  })

  it('drops points older than the cutoff', () => {
    const points = [{ t: 1000, val: 1 }, { t: 5000, val: 2 }]
    mergeLivePoint(points, { t: 6000, val: 3 }, 4000)
    expect(points.map(p => p.t)).toEqual([5000, 6000])
  })

  it('always keeps at least one point even if all are older than cutoff', () => {
    const points = [{ t: 1000, val: 1 }]
    mergeLivePoint(points, { t: 2000, val: 2 }, 10000)
    expect(points).toEqual([{ t: 2000, val: 2 }])
  })
})
