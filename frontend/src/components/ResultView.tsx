import { motion } from 'framer-motion';
import { 
  CheckCircle, 
  XCircle, 
  AlertTriangle, 
  HelpCircle,
  RotateCcw,
  ExternalLink,
  Quote,
  Sparkles
} from 'lucide-react';
import { FactCheckResult, Verdict, Claim, Source } from '../types';

interface ResultViewProps {
  result: FactCheckResult;
  processingTime?: number;
  onReset: () => void;
}

const verdictConfig: Record<Verdict, { 
  icon: typeof CheckCircle; 
  color: string; 
  bgColor: string; 
  label: string;
  description: string;
}> = {
  verified: { 
    icon: CheckCircle, 
    color: 'text-verdict-true', 
    bgColor: 'bg-verdict-true/20',
    label: 'Verified',
    description: 'This content appears to be accurate based on available evidence.'
  },
  false: { 
    icon: XCircle, 
    color: 'text-verdict-false', 
    bgColor: 'bg-verdict-false/20',
    label: 'False',
    description: 'This content contains significant inaccuracies or misinformation.'
  },
  misleading: { 
    icon: AlertTriangle, 
    color: 'text-verdict-mixed', 
    bgColor: 'bg-verdict-mixed/20',
    label: 'Misleading',
    description: 'This content is presented in a misleading way.'
  },
  partially_true: { 
    icon: AlertTriangle, 
    color: 'text-verdict-mixed', 
    bgColor: 'bg-verdict-mixed/20',
    label: 'Partially True',
    description: 'This content contains both accurate and inaccurate information.'
  },
  unverifiable: { 
    icon: HelpCircle, 
    color: 'text-verdict-unverified', 
    bgColor: 'bg-verdict-unverified/20',
    label: 'Unverifiable',
    description: 'We could not find enough evidence to verify this content.'
  },
  satire: { 
    icon: HelpCircle, 
    color: 'text-electric-cyan', 
    bgColor: 'bg-electric-cyan/20',
    label: 'Satire',
    description: 'This content appears to be satirical or parody.'
  },
  no_claim: { 
    icon: HelpCircle, 
    color: 'text-white/60', 
    bgColor: 'bg-white/10',
    label: 'No Claim Found',
    description: 'No factual claims were found to verify.'
  },
};

export default function ResultView({ result, processingTime, onReset }: ResultViewProps) {
  const config = verdictConfig[result.verdict];
  const VerdictIcon = config.icon;

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      className="w-full max-w-3xl"
    >
      {/* Verdict Header */}
      <motion.div
        initial={{ y: 20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        className="text-center mb-8"
      >
        <motion.div
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          transition={{ type: 'spring', delay: 0.1 }}
          className={`inline-flex w-24 h-24 rounded-2xl ${config.bgColor} items-center justify-center mb-4`}
        >
          <VerdictIcon className={`w-12 h-12 ${config.color}`} />
        </motion.div>

        <h2 className={`text-3xl font-bold ${config.color} mb-2`}>
          {config.label}
        </h2>
        
        <p className="text-white/60 max-w-lg mx-auto">
          {config.description}
        </p>

        {/* Confidence Score */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="mt-4 inline-flex items-center gap-2 px-4 py-2 glass rounded-full"
        >
          <Sparkles className="w-4 h-4 text-electric-cyan" />
          <span className="text-sm text-white/60">Confidence:</span>
          <span className="text-sm font-semibold text-white">
            {Math.round(result.confidence * 100)}%
          </span>
        </motion.div>
      </motion.div>

      {/* Summary */}
      <motion.div
        initial={{ y: 20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ delay: 0.2 }}
        className="glass rounded-2xl p-6 mb-6"
      >
        <div className="flex items-start gap-3">
          <Quote className="w-5 h-5 text-electric-blue flex-shrink-0 mt-1" />
          <p className="text-white/80 leading-relaxed">{result.summary}</p>
        </div>
      </motion.div>

      {/* Claims */}
      {result.claims && result.claims.length > 0 && (
        <motion.div
          initial={{ y: 20, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ delay: 0.3 }}
          className="mb-6"
        >
          <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-electric-blue" />
            Claims Analysis
          </h3>
          
          <div className="space-y-3">
            {result.claims.map((claim, index) => (
              <ClaimCard key={index} claim={claim} index={index} />
            ))}
          </div>
        </motion.div>
      )}

      {/* Sources */}
      {result.sources && result.sources.length > 0 && (
        <motion.div
          initial={{ y: 20, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ delay: 0.4 }}
          className="mb-8"
        >
          <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <ExternalLink className="w-5 h-5 text-electric-blue" />
            Sources Referenced
          </h3>
          
          <div className="grid gap-3 sm:grid-cols-2">
            {result.sources.map((source, index) => (
              <SourceCard key={index} source={source} index={index} />
            ))}
          </div>
        </motion.div>
      )}

      {/* Footer Actions */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.5 }}
        className="flex flex-col sm:flex-row items-center justify-center gap-4"
      >
        <button
          onClick={onReset}
          className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-electric-blue to-electric-cyan rounded-xl font-semibold text-white hover:shadow-lg hover:shadow-electric-blue/30 transition-all"
        >
          <RotateCcw className="w-5 h-5" />
          Check Another Post
        </button>

        {processingTime !== undefined && processingTime > 0 && (
          <span className="text-sm text-white/40">
            Analyzed in {processingTime.toFixed(1)}s
          </span>
        )}
      </motion.div>
    </motion.div>
  );
}

// Claim Card Component
function ClaimCard({ claim, index }: { claim: Claim; index: number }) {
  const config = verdictConfig[claim.verdict];
  const VerdictIcon = config.icon;

  return (
    <motion.div
      initial={{ x: -20, opacity: 0 }}
      animate={{ x: 0, opacity: 1 }}
      transition={{ delay: 0.1 * index }}
      className="glass rounded-xl p-4"
    >
      <div className="flex items-start gap-3">
        <div className={`flex-shrink-0 w-8 h-8 rounded-lg ${config.bgColor} flex items-center justify-center`}>
          <VerdictIcon className={`w-4 h-4 ${config.color}`} />
        </div>
        
        <div className="flex-1 min-w-0">
          <p className="text-white/90 font-medium mb-1">{claim.statement}</p>
          <p className="text-white/50 text-sm">{claim.explanation}</p>
        </div>
      </div>
    </motion.div>
  );
}

// Source Card Component
function SourceCard({ source, index }: { source: Source; index: number }) {
  return (
    <motion.a
      href={source.url || '#'}
      target="_blank"
      rel="noopener noreferrer"
      initial={{ y: 20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ delay: 0.05 * index }}
      className="glass rounded-xl p-4 hover:bg-white/10 transition-colors group"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <p className="text-white/90 font-medium text-sm truncate group-hover:text-electric-cyan transition-colors">
            {source.title}
          </p>
          <p className="text-white/40 text-xs truncate mt-1">
            {source.publisher}
          </p>
        </div>
        <ExternalLink className="w-4 h-4 text-white/30 group-hover:text-electric-cyan transition-colors flex-shrink-0" />
      </div>
      
      {source.credibility && (
        <div className="mt-2">
          <span className="text-xs px-2 py-1 rounded-full bg-electric-blue/20 text-electric-cyan">
            {source.credibility}
          </span>
        </div>
      )}
    </motion.a>
  );
}
