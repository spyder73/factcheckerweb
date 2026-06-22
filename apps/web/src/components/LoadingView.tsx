import { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { CheckResponse } from '../types';
import { Scan, Brain, Search, Scale, CheckCircle } from 'lucide-react';

interface LoadingViewProps {
  checkData: CheckResponse;
}

const steps = [
  { id: 'scraping', label: 'Extracting Content', icon: Scan, description: 'Analyzing the post...' },
  { id: 'analyzing', label: 'AI Analysis', icon: Brain, description: 'Processing with AI...' },
  { id: 'searching', label: 'Finding Sources', icon: Search, description: 'Searching for evidence...' },
  { id: 'evaluating', label: 'Evaluating Claims', icon: Scale, description: 'Checking truthfulness...' },
  { id: 'completed', label: 'Complete', icon: CheckCircle, description: 'Done!' },
];

export default function LoadingView({ checkData }: LoadingViewProps) {
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    // Map status to step index
    const statusMap: Record<string, number> = {
      pending: 0,
      scraping: 0,
      analyzing: 1,
      searching: 2,
      evaluating: 3,
      completed: 4,
    };

    const newIndex = statusMap[checkData.status] ?? 0;
    setCurrentStepIndex(newIndex);
    setProgress(checkData.progress);
  }, [checkData.status, checkData.progress]);

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      className="w-full max-w-lg text-center"
    >
      {/* Progress Ring */}
      <div className="relative w-48 h-48 mx-auto mb-8">
        {/* Background ring */}
        <svg className="w-full h-full -rotate-90" viewBox="0 0 100 100">
          <circle
            cx="50"
            cy="50"
            r="45"
            fill="none"
            stroke="rgba(255,255,255,0.1)"
            strokeWidth="3"
          />
          <motion.circle
            cx="50"
            cy="50"
            r="45"
            fill="none"
            stroke="url(#gradient)"
            strokeWidth="3"
            strokeLinecap="round"
            strokeDasharray={`${2 * Math.PI * 45}`}
            initial={{ strokeDashoffset: 2 * Math.PI * 45 }}
            animate={{ strokeDashoffset: 2 * Math.PI * 45 * (1 - progress / 100) }}
            transition={{ duration: 0.5, ease: 'easeOut' }}
          />
          <defs>
            <linearGradient id="gradient" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stopColor="#0066FF" />
              <stop offset="100%" stopColor="#00D4FF" />
            </linearGradient>
          </defs>
        </svg>

        {/* Center content */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <motion.div
            key={currentStepIndex}
            initial={{ scale: 0, rotate: -180 }}
            animate={{ scale: 1, rotate: 0 }}
            transition={{ type: 'spring', damping: 15 }}
            className="w-16 h-16 rounded-2xl bg-gradient-to-br from-electric-blue to-electric-cyan flex items-center justify-center mb-2"
          >
            {(() => {
              const StepIcon = steps[currentStepIndex]?.icon ?? Scan;
              return <StepIcon className="w-8 h-8 text-white" />;
            })()}
          </motion.div>
          <span className="text-3xl font-bold text-white">{Math.round(progress)}%</span>
        </div>

        {/* Glow effect */}
        <motion.div
          className="absolute inset-0 rounded-full bg-electric-blue/20 blur-xl -z-10"
          animate={{
            opacity: [0.3, 0.6, 0.3],
            scale: [0.8, 1, 0.8],
          }}
          transition={{
            duration: 2,
            repeat: Infinity,
            ease: 'easeInOut',
          }}
        />
      </div>

      {/* Current step label */}
      <motion.div
        key={checkData.status}
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        className="mb-8"
      >
        <h2 className="text-2xl font-bold text-white mb-2">
          {steps[currentStepIndex]?.label ?? 'Processing'}
        </h2>
        <p className="text-white/60">
          {checkData.currentStep || steps[currentStepIndex]?.description}
        </p>
      </motion.div>

      {/* Steps timeline */}
      <div className="flex justify-center gap-3">
        {steps.slice(0, -1).map((step, index) => {
          const isActive = index === currentStepIndex;
          const isCompleted = index < currentStepIndex;
          const StepIcon = step.icon;

          return (
            <motion.div
              key={step.id}
              initial={{ opacity: 0, scale: 0 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: index * 0.1 }}
              className="relative"
            >
              <div
                className={`w-12 h-12 rounded-xl flex items-center justify-center transition-all duration-300 ${
                  isActive
                    ? 'bg-gradient-to-br from-electric-blue to-electric-cyan glow-blue'
                    : isCompleted
                    ? 'bg-electric-blue/30'
                    : 'bg-white/5'
                }`}
              >
                <StepIcon
                  className={`w-5 h-5 transition-colors ${
                    isActive || isCompleted ? 'text-white' : 'text-white/30'
                  }`}
                />
              </div>

              {/* Connector line */}
              {index < steps.length - 2 && (
                <div
                  className={`absolute top-1/2 left-full w-3 h-0.5 -translate-y-1/2 transition-colors ${
                    isCompleted ? 'bg-electric-blue' : 'bg-white/10'
                  }`}
                />
              )}
            </motion.div>
          );
        })}
      </div>

      {/* Pulsing dots */}
      <motion.div
        className="flex justify-center gap-2 mt-8"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.5 }}
      >
        {[0, 1, 2].map((i) => (
          <motion.div
            key={i}
            className="w-2 h-2 bg-electric-blue rounded-full"
            animate={{
              scale: [1, 1.5, 1],
              opacity: [0.3, 1, 0.3],
            }}
            transition={{
              duration: 1,
              repeat: Infinity,
              delay: i * 0.2,
            }}
          />
        ))}
      </motion.div>
    </motion.div>
  );
}
