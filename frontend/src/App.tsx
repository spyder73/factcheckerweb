import { useState, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { ShieldCheck, Link, AlertCircle } from 'lucide-react';
import { startCheck, pollCheckStatus } from './api/factcheck';
import { CheckResponse } from './types';
import BackgroundEffects from './components/BackgroundEffects';
import LoadingView from './components/LoadingView';
import ResultView from './components/ResultView';

type AppState = 'idle' | 'loading' | 'result' | 'error';

function App() {
  const [url, setUrl] = useState('');
  const [appState, setAppState] = useState<AppState>('idle');
  const [checkData, setCheckData] = useState<CheckResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!url.trim()) return;

    setAppState('loading');
    setError(null);

    try {
      const response = await startCheck({ url: url.trim() });
      setCheckData(response);

      // Poll for updates
      pollCheckStatus(response.id, (data) => {
        setCheckData(data);
        
        if (data.status === 'completed') {
          setAppState('result');
        } else if (data.status === 'error') {
          setError(data.error || 'An error occurred');
          setAppState('error');
        }
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start fact-check');
      setAppState('error');
    }
  }, [url]);

  const handleReset = useCallback(() => {
    setUrl('');
    setAppState('idle');
    setCheckData(null);
    setError(null);
  }, []);

  return (
    <div className="min-h-screen animated-bg relative overflow-hidden">
      <BackgroundEffects />
      
      <div className="relative z-10 min-h-screen flex flex-col">
        {/* Header */}
        <header className="p-6">
          <motion.div 
            className="flex items-center gap-3 cursor-pointer"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            onClick={handleReset}
          >
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-electric-blue to-electric-cyan flex items-center justify-center">
              <ShieldCheck className="w-6 h-6 text-white" />
            </div>
            <span className="text-xl font-bold gradient-text">FactChecker</span>
          </motion.div>
        </header>

        {/* Main Content */}
        <main className="flex-1 flex items-center justify-center px-4 py-8">
          <AnimatePresence mode="wait">
            {appState === 'idle' && (
              <IdleView 
                key="idle"
                url={url}
                setUrl={setUrl}
                onSubmit={handleSubmit}
              />
            )}

            {appState === 'loading' && checkData && (
              <LoadingView 
                key="loading"
                checkData={checkData}
              />
            )}

            {appState === 'result' && checkData?.result && (
              <ResultView 
                key="result"
                result={checkData.result}
                processingTime={checkData.processingTime}
                onReset={handleReset}
              />
            )}

            {appState === 'error' && (
              <ErrorView 
                key="error"
                error={error || 'Unknown error'}
                onRetry={handleReset}
              />
            )}
          </AnimatePresence>
        </main>

        {/* Footer */}
        <footer className="p-6 text-center">
          <p className="text-sm text-white/40">
            Powered by AI • Verify before you share
          </p>
        </footer>
      </div>
    </div>
  );
}

// Idle View Component
interface IdleViewProps {
  url: string;
  setUrl: (url: string) => void;
  onSubmit: (e: React.FormEvent) => void;
}

function IdleView({ url, setUrl, onSubmit }: IdleViewProps) {
  const [isFocused, setIsFocused] = useState(false);

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      className="w-full max-w-2xl"
    >
      <div className="text-center mb-12">
        <motion.div
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          transition={{ type: 'spring', delay: 0.2 }}
          className="inline-block mb-6"
        >
          <div className="relative">
            <div className="absolute inset-0 bg-electric-blue/30 blur-3xl rounded-full" />
            <div className="relative w-24 h-24 rounded-2xl bg-gradient-to-br from-electric-blue to-electric-cyan flex items-center justify-center glow-blue">
              <ShieldCheck className="w-12 h-12 text-white" />
            </div>
          </div>
        </motion.div>

        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          className="text-4xl md:text-5xl font-bold mb-4"
        >
          <span className="gradient-text">Verify</span>
          <span className="text-white"> Before You Share</span>
        </motion.h1>

        <motion.p
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
          className="text-lg text-white/60 max-w-md mx-auto"
        >
          Paste any social media post link and we'll analyze it for accuracy using AI
        </motion.p>
      </div>

      <motion.form
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.5 }}
        onSubmit={onSubmit}
        className="space-y-4"
      >
        <div className={`relative rounded-2xl transition-all duration-300 ${
          isFocused ? 'glow-blue' : ''
        }`}>
          <div className="absolute inset-0 bg-gradient-to-r from-electric-blue to-electric-cyan rounded-2xl opacity-50 blur-sm -z-10" />
          
          <div className="glass rounded-2xl p-2">
            <div className="flex items-center gap-3 px-4">
              <Link className="w-5 h-5 text-electric-blue" />
              <input
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                onFocus={() => setIsFocused(true)}
                onBlur={() => setIsFocused(false)}
                placeholder="Paste Instagram, Twitter, TikTok, or any social media link..."
                className="flex-1 bg-transparent py-4 text-white placeholder-white/40 outline-none text-lg"
              />
            </div>
          </div>
        </div>

        <motion.button
          type="submit"
          disabled={!url.trim()}
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          className="w-full py-4 px-6 bg-gradient-to-r from-electric-blue to-electric-cyan rounded-xl font-semibold text-white text-lg disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-300 hover:shadow-lg hover:shadow-electric-blue/30"
        >
          Check Facts
        </motion.button>
      </motion.form>

      {/* Supported platforms */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.7 }}
        className="mt-12 text-center"
      >
        <p className="text-sm text-white/40 mb-4">Supports all major platforms</p>
        <div className="flex justify-center gap-6 text-white/30">
          {['Instagram', 'Twitter/X', 'TikTok', 'Facebook', 'YouTube'].map((platform) => (
            <span key={platform} className="text-sm hover:text-white/60 transition-colors cursor-default">
              {platform}
            </span>
          ))}
        </div>
      </motion.div>
    </motion.div>
  );
}

// Error View Component
interface ErrorViewProps {
  error: string;
  onRetry: () => void;
}

function ErrorView({ error, onRetry }: ErrorViewProps) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      className="text-center"
    >
      <div className="w-20 h-20 rounded-full bg-verdict-false/20 flex items-center justify-center mx-auto mb-6">
        <AlertCircle className="w-10 h-10 text-verdict-false" />
      </div>

      <h2 className="text-2xl font-bold text-white mb-2">Something went wrong</h2>
      <p className="text-white/60 mb-8 max-w-md">{error}</p>

      <button
        onClick={onRetry}
        className="px-8 py-3 bg-gradient-to-r from-electric-blue to-electric-cyan rounded-xl font-semibold text-white hover:shadow-lg hover:shadow-electric-blue/30 transition-all"
      >
        Try Again
      </button>
    </motion.div>
  );
}

export default App;
