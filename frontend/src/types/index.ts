// Types for the FactChecker API

export type CheckStatus = 'pending' | 'processing' | 'completed' | 'error';

export type Verdict = 
  | 'verified' 
  | 'false' 
  | 'misleading' 
  | 'partially_true' 
  | 'unverifiable' 
  | 'satire'
  | 'no_claim';

export type Stance = 'supports' | 'contradicts' | 'neutral' | 'context';

export interface CheckRequest {
  url: string;
  caption?: string;
}

export interface CheckResponse {
  id: string;
  status: CheckStatus;
  progress: number;
  currentStep: string;
  result?: FactCheckResult;
  error?: string;
  processingTime: number;
  createdAt: string;
}

export interface FactCheckResult {
  verdict: Verdict;
  confidence: number;
  summary: string;
  claims: Claim[];
  sources: Source[];
  mediaAnalysis: MediaInfo[];
  originalContent: ContentInfo;
}

export interface Claim {
  id: string;
  statement: string;
  verdict: Verdict;
  explanation: string;
  proArguments: string[];
  contraArguments: string[];
  sourceIds?: string[];
}

export interface Source {
  id: string;
  title: string;
  url?: string;
  publisher: string;
  publishedAt?: string;
  credibility: string;
  excerpt?: string;
  stance: Stance;
}

export interface MediaInfo {
  id: string;
  type: 'image' | 'video_frame';
  url?: string;
  description: string;
  elements: string[];
  textFound?: string;
}

export interface ContentInfo {
  platform: string;
  url: string;
  caption: string;
  mediaUrls: string[];
  author?: string;
  postedAt?: string;
}

export interface ProgressUpdate {
  step: string;
  progress: number;
  message: string;
}
