// Home / paste-and-check screen. Picks up shared content from the
// share extension (expo-share-intent) automatically; user can also paste
// or type a URL. Submitting POSTs /api/check and navigates into the
// streaming verdict screen.

import { useEffect, useState } from 'react'
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native'
import { router } from 'expo-router'
import { useShareIntent } from 'expo-share-intent'
import { api } from '../lib/api'
import { COLORS } from '../lib/theme'
import { ApiException } from '@alethea/shared-types'

interface StartResp {
  id: string
}

export default function Home() {
  const [text, setText] = useState('')
  const [busy, setBusy] = useState(false)
  const { hasShareIntent, shareIntent, resetShareIntent } = useShareIntent()

  useEffect(() => {
    if (hasShareIntent && shareIntent) {
      const incoming =
        shareIntent.webUrl ??
        shareIntent.text ??
        (shareIntent.files?.[0]?.path ?? '')
      if (incoming) setText(incoming)
      resetShareIntent()
    }
  }, [hasShareIntent, shareIntent, resetShareIntent])

  async function onSubmit() {
    const trimmed = text.trim()
    if (!trimmed) return
    setBusy(true)
    try {
      const body = trimmed.startsWith('http')
        ? { url: trimmed }
        : { text: trimmed }
      const r = await api<StartResp>('/api/check', { method: 'POST', body })
      router.push(`/check/${r.id}`)
    } catch (e) {
      if (e instanceof ApiException) {
        Alert.alert('Could not start check', e.message)
      } else {
        Alert.alert('Could not start check', 'Unknown error.')
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      style={styles.root}
    >
      <View style={styles.body}>
        <Text style={styles.eyebrow}>PASTE ANYTHING · GET A VERDICT YOU CAN TRACE</Text>
        <Text style={styles.h1}>What do you want to check?</Text>

        <TextInput
          style={styles.input}
          value={text}
          onChangeText={setText}
          multiline
          placeholder="Paste a link, screenshot caption, or claim…"
          placeholderTextColor={COLORS.fgSubtle}
          editable={!busy}
        />

        <Pressable
          accessibilityRole="button"
          onPress={onSubmit}
          disabled={busy || !text.trim()}
          style={({ pressed }) => [
            styles.cta,
            (busy || !text.trim()) && styles.ctaDisabled,
            pressed && { opacity: 0.85 },
          ]}
        >
          {busy
            ? <ActivityIndicator color={COLORS.accentFg} />
            : <Text style={styles.ctaLabel}>Check it</Text>}
        </Pressable>

        <View style={styles.footerRow}>
          <Pressable onPress={() => router.push('/history')}>
            <Text style={styles.link}>History</Text>
          </Pressable>
          <Pressable onPress={() => router.push('/settings')}>
            <Text style={styles.link}>Settings</Text>
          </Pressable>
        </View>
      </View>
    </KeyboardAvoidingView>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  body: { flex: 1, padding: 20, justifyContent: 'center' },
  eyebrow: { color: COLORS.accent, fontSize: 11, letterSpacing: 1.2, marginBottom: 12 },
  h1: { color: COLORS.fg, fontSize: 28, fontWeight: '700', marginBottom: 24 },
  input: {
    minHeight: 120,
    color: COLORS.fg,
    backgroundColor: COLORS.bgElevated,
    borderColor: COLORS.border,
    borderWidth: 1,
    borderRadius: 12,
    padding: 14,
    fontSize: 16,
    textAlignVertical: 'top',
  },
  cta: {
    marginTop: 16,
    height: 52,
    borderRadius: 12,
    backgroundColor: COLORS.accent,
    alignItems: 'center',
    justifyContent: 'center',
  },
  ctaDisabled: { opacity: 0.5 },
  ctaLabel: { color: COLORS.accentFg, fontSize: 16, fontWeight: '600' },
  footerRow: { flexDirection: 'row', justifyContent: 'space-between', marginTop: 24 },
  link: { color: COLORS.fgMuted, fontSize: 14 },
})
