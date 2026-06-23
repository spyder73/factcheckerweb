import { useEffect, useState } from 'react'
import { ActivityIndicator, FlatList, Pressable, StyleSheet, Text, View } from 'react-native'
import { router } from 'expo-router'
import { api } from '../lib/api'
import { COLORS, VERDICT_COLORS, VERDICT_LABEL } from '../lib/theme'
import type { CheckRow, Verdict } from '@alethea/shared-types'

interface HistoryResp { checks: (CheckRow & { overall_verdict?: Verdict })[] }

export default function History() {
  const [items, setItems] = useState<HistoryResp['checks'] | null>(null)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    void api<HistoryResp>('/api/me/checks')
      .then((r) => setItems(r.checks ?? []))
      .catch((e) => setErr(e instanceof Error ? e.message : 'load failed'))
  }, [])

  if (err) return <View style={styles.center}><Text style={styles.err}>{err}</Text></View>
  if (!items) return <View style={styles.center}><ActivityIndicator color={COLORS.accent} /></View>
  if (items.length === 0) {
    return (
      <View style={styles.center}>
        <Text style={styles.empty}>No checks yet.</Text>
        <Text style={styles.emptySub}>Paste something on the home screen to get started.</Text>
      </View>
    )
  }

  return (
    <FlatList
      style={styles.list}
      data={items}
      keyExtractor={(it) => it.id}
      renderItem={({ item }) => {
        const v = item.overall_verdict ?? 'unverifiable'
        const color = VERDICT_COLORS[v]
        return (
          <Pressable
            onPress={() => router.push(`/check/${item.id}`)}
            style={({ pressed }) => [styles.row, pressed && { opacity: 0.7 }]}
          >
            <Text style={[styles.rowLabel, { color }]}>{VERDICT_LABEL[v]}</Text>
            <Text style={styles.rowText} numberOfLines={2}>
              {item.input_url || item.input_caption || '(no input)'}
            </Text>
            <Text style={styles.rowDate}>{new Date(item.created_at).toLocaleString()}</Text>
          </Pressable>
        )
      }}
    />
  )
}

const styles = StyleSheet.create({
  list: { padding: 16 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  err: { color: VERDICT_COLORS.false },
  empty: { color: COLORS.fg, fontSize: 18, fontWeight: '600' },
  emptySub: { color: COLORS.fgMuted, marginTop: 8, textAlign: 'center' },
  row: {
    padding: 14, marginBottom: 10, borderRadius: 12, borderWidth: 1,
    borderColor: COLORS.border, backgroundColor: COLORS.bgElevated,
  },
  rowLabel: { fontSize: 11, fontWeight: '700', letterSpacing: 1 },
  rowText: { color: COLORS.fg, marginTop: 4, fontSize: 14 },
  rowDate: { color: COLORS.fgSubtle, marginTop: 6, fontSize: 12 },
})
