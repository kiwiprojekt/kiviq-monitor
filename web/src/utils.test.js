import { describe, it, expect, beforeEach } from 'vitest'
import { formatBytes, formatRate, textColor, barColor, tempColor, formatUptime, generateToken, setCredentials, getCredentials, clearCredentials, getAuth } from './utils.js'

describe('formatBytes', () => {
  it('returns "0 B" for falsy values', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(null)).toBe('0 B')
    expect(formatBytes(undefined)).toBe('0 B')
  })

  it('formats bytes', () => {
    expect(formatBytes(500)).toBe('500.0 B')
  })

  it('formats kilobytes', () => {
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1536)).toBe('1.5 KB')
  })

  it('formats megabytes', () => {
    expect(formatBytes(1048576)).toBe('1.0 MB')
  })

  it('formats gigabytes', () => {
    expect(formatBytes(1073741824)).toBe('1.0 GB')
  })

  it('formats terabytes', () => {
    expect(formatBytes(1099511627776)).toBe('1.0 TB')
  })

  it('uses correct unit boundaries', () => {
    expect(formatBytes(1023)).toBe('1023.0 B')
    expect(formatBytes(1025)).toBe('1.0 KB')
  })
})

describe('formatRate', () => {
  it('formats bytes per second', () => {
    expect(formatRate(500)).toBe('500 B/s')
  })

  it('formats kilobytes per second', () => {
    expect(formatRate(2048)).toBe('2.0 KB/s')
  })

  it('formats megabytes per second', () => {
    expect(formatRate(2097152)).toBe('2.0 MB/s')
  })

  it('formats gigabytes per second', () => {
    expect(formatRate(2147483648)).toBe('2.00 GB/s')
  })

  it('boundary at 1024', () => {
    expect(formatRate(1023)).toBe('1023 B/s')
    expect(formatRate(1024)).toBe('1.0 KB/s')
  })
})

describe('textColor', () => {
  it('returns danger for > 90', () => {
    expect(textColor(95)).toBe('text-danger')
    expect(textColor(100)).toBe('text-danger')
  })

  it('returns warning for > 70', () => {
    expect(textColor(75)).toBe('text-warning')
    expect(textColor(90)).toBe('text-warning')
  })

  it('returns success for <= 70', () => {
    expect(textColor(70)).toBe('text-success')
    expect(textColor(50)).toBe('text-success')
    expect(textColor(0)).toBe('text-success')
  })
})

describe('barColor', () => {
  it('returns danger for > 90', () => {
    expect(barColor(95)).toBe('bg-danger')
  })

  it('returns warning for > 70', () => {
    expect(barColor(80)).toBe('bg-warning')
  })

  it('returns success for <= 70', () => {
    expect(barColor(70)).toBe('bg-success')
    expect(barColor(30)).toBe('bg-success')
  })
})

describe('tempColor', () => {
  it('returns danger for >= 90', () => {
    expect(tempColor(90)).toBe('text-danger')
    expect(tempColor(100)).toBe('text-danger')
  })

  it('returns warning for >= 70', () => {
    expect(tempColor(70)).toBe('text-warning')
    expect(tempColor(89)).toBe('text-warning')
  })

  it('returns success for < 70', () => {
    expect(tempColor(69)).toBe('text-success')
    expect(tempColor(40)).toBe('text-success')
  })
})

describe('formatUptime', () => {
  it('formats minutes', () => {
    expect(formatUptime(300)).toBe('5m')
  })

  it('formats hours and minutes', () => {
    expect(formatUptime(3660)).toBe('1h 1m')
  })

  it('formats days and hours', () => {
    expect(formatUptime(90000)).toBe('1d 1h')
  })

  it('handles zero', () => {
    expect(formatUptime(0)).toBe('0m')
  })

  it('handles undefined', () => {
    expect(formatUptime(undefined)).toBe('0m')
  })

  it('exact hours shows 0 minutes', () => {
    expect(formatUptime(7200)).toBe('2h 0m')
  })

  it('exact days shows 0 hours', () => {
    expect(formatUptime(172800)).toBe('2d 0h')
  })
})

describe('generateToken', () => {
  it('returns hex string of requested length', () => {
    const token = generateToken(16)
    expect(token).toMatch(/^[0-9a-f]{32}$/)
  })

  it('defaults to 32 bytes', () => {
    const token = generateToken()
    expect(token).toMatch(/^[0-9a-f]{64}$/)
  })

  it('generates unique tokens', () => {
    const a = generateToken()
    const b = generateToken()
    expect(a).not.toBe(b)
  })
})

describe('setCredentials / getCredentials', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('round-trips credentials through sessionStorage', () => {
    setCredentials('admin', 'secret')
    const { user, pass } = getCredentials()
    expect(user).toBe('admin')
    expect(pass).toBe('secret')
  })

  it('returns empty strings when no credentials stored', () => {
    const { user, pass } = getCredentials()
    expect(user).toBe('')
    expect(pass).toBe('')
  })

  it('clearCredentials removes stored credentials', () => {
    setCredentials('admin', 'secret')
    clearCredentials()
    const { user, pass } = getCredentials()
    expect(user).toBe('')
    expect(pass).toBe('')
  })
})

describe('getAuth', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('returns base64 encoded user:pass', () => {
    setCredentials('admin', 'secret')
    expect(getAuth()).toBe(btoa('admin:secret'))
  })

  it('returns base64 of empty credentials', () => {
    expect(getAuth()).toBe(btoa(':'))
  })
})
