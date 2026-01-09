import { useState } from 'react';
import type { Messages } from '../types';

export const useMessages = () => {
  const [messages, setMessages] = useState<Messages>({});

  const setMessage = (key: string, message: string | null, type: 'info' | 'success' | 'error' = 'info') => {
    if (message === null) {
      setMessages(prev => {
        const newMessages = { ...prev };
        delete newMessages[key];
        return newMessages;
      });
    } else {
      setMessages(prev => ({ ...prev, [key]: { message, type } }));
    }
  };

  const clearMessage = (key: string) => {
    setMessage(key, null);
  };

  const clearAllMessages = () => {
    setMessages({});
  };

  return {
    messages,
    setMessage,
    clearMessage,
    clearAllMessages,
  };
};