/// <reference types="vite/client" />
import { CheckRequest, CheckResponse } from '../types';

const API_BASE = import.meta.env.DEV ? 'http://localhost:8080' : '';

export async function startCheck(request: CheckRequest): Promise<CheckResponse> {
  const response = await fetch(`${API_BASE}/api/check`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    throw new Error('Failed to start fact-check');
  }

  return response.json();
}

export async function getCheckStatus(id: string): Promise<CheckResponse> {
  const response = await fetch(`${API_BASE}/api/check/${id}`);

  if (!response.ok) {
    throw new Error('Failed to get check status');
  }

  return response.json();
}

export function subscribeToProgress(
  id: string, 
  onProgress: (data: CheckResponse) => void,
  onComplete: (data: CheckResponse) => void,
  onError: (error: Error) => void
): () => void {
  const eventSource = new EventSource(`${API_BASE}/api/check/${id}/stream`);

  eventSource.addEventListener('status', (event) => {
    const statusData = JSON.parse(event.data) as CheckResponse;
    onProgress(statusData);
  });

  eventSource.addEventListener('progress', () => {
    // Get full status on progress update
    getCheckStatus(id).then(onProgress).catch(console.error);
  });

  eventSource.addEventListener('complete', (event) => {
    const completeData = JSON.parse(event.data) as CheckResponse;
    onComplete(completeData);
    eventSource.close();
  });

  eventSource.onerror = () => {
    onError(new Error('Connection lost'));
    eventSource.close();
  };

  return () => eventSource.close();
}

// Polling fallback if SSE doesn't work
export async function pollCheckStatus(
  id: string,
  onUpdate: (data: CheckResponse) => void,
  interval = 1000
): Promise<void> {
  const poll = async () => {
    try {
      const status = await getCheckStatus(id);
      onUpdate(status);

      if (status.status === 'completed' || status.status === 'error') {
        return;
      }

      setTimeout(poll, interval);
    } catch (error) {
      console.error('Polling error:', error);
      setTimeout(poll, interval * 2);
    }
  };

  poll();
}
