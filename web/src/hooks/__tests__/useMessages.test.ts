import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useMessages } from '../useMessages';

describe('useMessages', () => {
  it('should initialize with empty messages', () => {
    const { result } = renderHook(() => useMessages());
    
    expect(result.current.messages).toEqual({});
  });

  it('should set a message', () => {
    const { result } = renderHook(() => useMessages());
    
    act(() => {
      result.current.setMessage('test', 'Test message', 'success');
    });

    expect(result.current.messages).toEqual({
      test: { message: 'Test message', type: 'success' }
    });
  });

  it('should set multiple messages', () => {
    const { result } = renderHook(() => useMessages());
    
    act(() => {
      result.current.setMessage('success', 'Success message', 'success');
      result.current.setMessage('error', 'Error message', 'error');
      result.current.setMessage('info', 'Info message', 'info');
    });

    expect(result.current.messages).toEqual({
      success: { message: 'Success message', type: 'success' },
      error: { message: 'Error message', type: 'error' },
      info: { message: 'Info message', type: 'info' },
    });
  });

  it('should update existing message', () => {
    const { result } = renderHook(() => useMessages());
    
    act(() => {
      result.current.setMessage('test', 'First message', 'info');
    });

    expect(result.current.messages.test).toEqual({
      message: 'First message',
      type: 'info'
    });

    act(() => {
      result.current.setMessage('test', 'Updated message', 'error');
    });

    expect(result.current.messages.test).toEqual({
      message: 'Updated message',
      type: 'error'
    });
  });

  it('should clear a specific message', () => {
    const { result } = renderHook(() => useMessages());
    
    act(() => {
      result.current.setMessage('test1', 'Message 1', 'info');
      result.current.setMessage('test2', 'Message 2', 'success');
    });

    expect(Object.keys(result.current.messages)).toHaveLength(2);

    act(() => {
      result.current.clearMessage('test1');
    });

    expect(result.current.messages).toEqual({
      test2: { message: 'Message 2', type: 'success' }
    });
  });

  it('should clear message by setting to null', () => {
    const { result } = renderHook(() => useMessages());
    
    act(() => {
      result.current.setMessage('test', 'Test message', 'info');
    });

    expect(result.current.messages.test).toBeDefined();

    act(() => {
      result.current.setMessage('test', null);
    });

    expect(result.current.messages.test).toBeUndefined();
  });

  it('should clear all messages', () => {
    const { result } = renderHook(() => useMessages());
    
    act(() => {
      result.current.setMessage('test1', 'Message 1', 'info');
      result.current.setMessage('test2', 'Message 2', 'success');
      result.current.setMessage('test3', 'Message 3', 'error');
    });

    expect(Object.keys(result.current.messages)).toHaveLength(3);

    act(() => {
      result.current.clearAllMessages();
    });

    expect(result.current.messages).toEqual({});
  });

  it('should default to info type when type not specified', () => {
    const { result } = renderHook(() => useMessages());
    
    act(() => {
      result.current.setMessage('test', 'Test message');
    });

    expect(result.current.messages.test).toEqual({
      message: 'Test message',
      type: 'info'
    });
  });
});