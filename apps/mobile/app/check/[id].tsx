// Verdict / streaming-progress screen. Polls /api/check/{id} until done.
//
// We POLL instead of SSE because React Native's fetch doesn't expose the
// streaming body in a way that's reliable across iOS and Android. The
// trade-off is a slightly less smooth progress indicator. Switch to a
// websocket transport later if it matters.

import { useEffect, useState } from 'react'
import { ActivityIndicator, ScrollView, StyleSheet, Text, View } from 'react-native'
import { useLocalSearchParams } from 'expo-router'
import { api } from '../../lib/api'
import { COLORS, VERDICT_COLORS, VERDICT_LABEL, VERDICT_ACTION_LINE } from '../../lib/theme'
import type { CheckRow, CheckDonePayload } from '@alethea/shared-types'

type Combined = CheckRow & Partial<CheckDonePayload>

export default function CheckScreen() {
  const { id } = useLocalSearchParams<{ id: string }>()
  const [data, setData] = useState<Combined | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | null = null

    async function tick() {
      try {
        const r = await api<Combined>(`/api/check/${id}`)
        if (cancelled) return
        setData(r)
        if (r.status !== 'completed' && r.status !== 'error' && r.status !== 'timeout') {
          timer = setTimeout(tick, 1500)
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'fetch failed')
      }
    }
    void tick()
    return () => { cancelled = true; if (timer) clearTimeout(timer) }
  }, [id])

  if (error) {
    return <Centered><Text style={styles.errText}>{error}</Text></Centered>
  }
  if (!data) {
    return <Centered><ActivityIndicator color={COLORS.accent} /></Centered>
  }
  if (data.status !== 'completed') {
    return (
      <Centered>
        <ActivityIndicator color={COLORS.accent} />
        <Text style={styles.statusText}>Working — {data.status}…</Text>
      </Centered>
    )
  }

  const verdict = data.overall_verdict ?? 'unverifiable'
  const color = VERDICT_COLORS[verdict]
  const label = VERDICT_LABEL[verdict]
  const actionLine = VERDICT_ACTION_LINE[verdict]

  return (
    <ScrollView contentContainerStyle={styles.scroll}>
      <View style={[styles.verdictPill, { borderColor: color }]}>
        <View style={[styles.dot, { backgroundColor: color }]} />
        <Text style={[styles.verdictLabel, { color }]}>{label}</Text>
      </View>
      <Text style={[styles.actionLine, { color }]}>{actionLine}</Text>
      <Text style={styles.summary}>{data.summary}</Text>

      {data.claims?.map((c) => (
        <View key={c.claim.id} style={styles.claim}>
          <Text style={styles.claimText}>{c.claim.canonical_text}</Text>
          <Text style={[styles.claimVerdict, { color: VERDICT_COLORS[c.final.verdict] }]}>
            {VERDICT_LABEL[c.final.verdict]} · {Math.round(c.final.confidence * 100)}%
          </Text>
          <Text style={styles.reasoning}>{c.final.reasoning}</Text>
          {c.final.cited_urls?.length > 0 && (
            <View style={{ marginTop: 8 }}>
              {c.final.cited_urls.map((u, i) => (
                <Text key={i} style={styles.url} numberOfLines={1}>· {u}</Text>
              ))}
            </View>
          )}
        </View>
      ))}
    </ScrollView>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return <View style={styles.centered}>{children}</View>
}

const styles = StyleSheet.create({
  scroll: { padding: 20 },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  errText: { color: VERDICT_COLORS.false, padding: 20, textAlign: 'center' },
  statusText: { color: COLORS.fgMuted, marginTop: 12 },
  verdictPill: {
    flexDirection: 'row', alignItems: 'center', alignSelf: 'flex-start',
    paddingHorizontal: 14, paddingVertical: 8, borderWidth: 1, borderRadius: 999, gap: 8,
  },
  dot: { width: 10, height: 10, borderRadius: 5 },
  verdictLabel: { fontSize: 13, fontWeight: '700', letterSpacing: 1 },
  actionLine: { fontSize: 18, fontWeight: '600', marginTop: 12 },
  summary: { color: COLORS.fg, fontSize: 16, lineHeight: 24, marginTop: 16 },
  claim: {
    marginTop: 24, padding: 16, borderColor: COLORS.border, borderWidth: 1, borderRadius: 12,
    backgroundColor: COLORS.bgElevated,
  },
  claimText: { color: COLORS.fg, fontSize: 15, fontWeight: '500' },
  claimVerdict: { fontSize: 12, fontWeight: '700', letterSpacing: 1, marginTop: 8 },
  reasoning: { color: COLORS.fgMuted, fontSize: 14, lineHeight: 21, marginTop: 8 },
  url: { color: COLORS.accent, fontSize: 12 },
})
