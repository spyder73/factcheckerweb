import { useEffect, useState } from 'react'
import { Alert, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native'
import { router } from 'expo-router'
import { api, clearAuth, getSessionToken } from '../lib/api'
import { COLORS } from '../lib/theme'
import type { MeResponse } from '@alethea/shared-types'

export default function Settings() {
  const [me, setMe] = useState<MeResponse | null>(null)
  const [hasToken, setHasToken] = useState<boolean | null>(null)

  useEffect(() => {
    void (async () => {
      setHasToken(!!(await getSessionToken()))
      try {
        const r = await api<MeResponse>('/auth/me')
        setMe(r)
      } catch { /* not signed in */ }
    })()
  }, [])

  async function logout() {
    try { await api('/auth/logout', { method: 'POST' }) } catch { /* ignore */ }
    await clearAuth()
    router.replace('/login')
  }

  async function deleteAccount() {
    Alert.alert(
      'Delete account?',
      'This permanently erases your account, history, and BYOK keys. Active Stripe subscriptions are NOT automatically canceled — open the Customer Portal first if you have one.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete forever', style: 'destructive', onPress: async () => {
            try {
              await api('/api/me/data-delete', { method: 'POST', body: { confirmation: 'DELETE' } })
              await clearAuth()
              router.replace('/login')
            } catch (e) {
              Alert.alert('Could not delete', e instanceof Error ? e.message : 'Unknown error')
            }
          },
        },
      ],
    )
  }

  return (
    <ScrollView contentContainerStyle={styles.body}>
      <Section title="Account">
        {me ? (
          <>
            <Row label="Email" value={me.user.email} />
            <Row label="Plan" value={me.user.plan.toUpperCase()} />
          </>
        ) : hasToken === false ? (
          <Pressable onPress={() => router.push('/login')} style={styles.row}>
            <Text style={styles.rowLink}>Sign in</Text>
          </Pressable>
        ) : (
          <Text style={styles.rowMuted}>Loading…</Text>
        )}
      </Section>

      <Section title="Privacy">
        <Pressable
          onPress={() => Alert.alert('Coming soon', 'Tap and we hit /api/me/data-export — implemented on the backend; mobile UI for the download is on the roadmap.')}
          style={styles.row}>
          <Text style={styles.rowLink}>Export my data</Text>
          <Text style={styles.rowSub}>GDPR Article 15 · JSON download</Text>
        </Pressable>
        {me && (
          <Pressable onPress={deleteAccount} style={styles.row}>
            <Text style={[styles.rowLink, { color: '#f85149' }]}>Delete my account</Text>
            <Text style={styles.rowSub}>GDPR Article 17 · permanent</Text>
          </Pressable>
        )}
      </Section>

      {me && (
        <Section title="Session">
          <Pressable onPress={logout} style={styles.row}>
            <Text style={styles.rowLink}>Sign out</Text>
          </Pressable>
        </Section>
      )}

      <Section title="About">
        <Row label="Build" value="0.1.0 · scaffold" />
        <Row label="Source" value="github.com/alethea-app" />
      </Section>
    </ScrollView>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <Text style={styles.sectionTitle}>{title}</Text>
      <View style={styles.sectionBody}>{children}</View>
    </View>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.row}>
      <Text style={styles.rowLabel}>{label}</Text>
      <Text style={styles.rowValue}>{value}</Text>
    </View>
  )
}

const styles = StyleSheet.create({
  body: { padding: 16 },
  section: { marginBottom: 24 },
  sectionTitle: { color: COLORS.fgSubtle, fontSize: 11, letterSpacing: 1.4, marginBottom: 8 },
  sectionBody: { backgroundColor: COLORS.bgElevated, borderColor: COLORS.border, borderWidth: 1, borderRadius: 12 },
  row: { padding: 14, borderBottomColor: COLORS.borderSubtle, borderBottomWidth: 1 },
  rowLabel: { color: COLORS.fgMuted, fontSize: 12 },
  rowValue: { color: COLORS.fg, fontSize: 15, marginTop: 2 },
  rowLink: { color: COLORS.fg, fontSize: 15, fontWeight: '500' },
  rowSub: { color: COLORS.fgSubtle, fontSize: 12, marginTop: 2 },
  rowMuted: { color: COLORS.fgMuted, padding: 14 },
})
