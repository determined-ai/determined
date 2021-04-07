import { EventEmitter } from 'events';

import { useCallback, useEffect } from 'react';

export enum KeyEvent {
  KeyUp = 'KeyUp'
}

export const keyEmitter = new EventEmitter();

let listenerCount = 0;

const useKeyTracker = (): void => {
  const handleKeyUp = useCallback((e: KeyboardEvent) => {
    if (e.target) {
      const element = e.target as Element;
      if ([ 'input', 'textarea' ].includes(element.tagName.toLowerCase())) return;
      if (element.getAttribute('contenteditable')) return;
    }
    keyEmitter.emit(KeyEvent.KeyUp, e);
  }, []);

  useEffect(() => {
    if (listenerCount !== 0) return;

    listenerCount++;
    document.body.addEventListener('keyup', handleKeyUp);

    return () => {
      if (listenerCount === 0) return;
      document.body.removeEventListener('keyup', handleKeyUp);
      listenerCount--;
    };
  }, [ handleKeyUp ]);
};

export default useKeyTracker;
