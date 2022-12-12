import OmnibarNpm from 'omnibar';
import React, { useCallback, useEffect, useState } from 'react';

import { KeyCode, keyEmitter, KeyEvent } from 'hooks/useKeyTracker';
import * as Tree from 'omnibar/tree-extension/index';
import TreeNode from 'omnibar/tree-extension/TreeNode';
import { BaseNode } from 'omnibar/tree-extension/types';
import { isTreeNode } from 'omnibar/tree-extension/utils';
import handleError from 'utils/error';

import css from './Omnibar.module.scss';

/**
 * Ideally we wouldn't need to access the element like this.
 * A potential option is use the value prop in combinatio with encoding the tree path into
 * the options returned by the tree extension.
 */
const omnibarInput = () =>
  document.querySelector('#omnibar input[type="text"]') as HTMLInputElement | null;

const Omnibar: React.FC = () => {
  const [showing, setShowing] = useState(false);

  useEffect(() => {
    const keyDownListener = (e: KeyboardEvent) => {
      if (e.code === KeyCode.Space && e.ctrlKey) {
        setShowing((showing) => !showing);
      } else if (showing && e.code === KeyCode.Escape) {
        setShowing(false);
      }
    };

    keyEmitter.on(KeyEvent.KeyDown, keyDownListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyDown, keyDownListener);
    };
  }, [showing]);

  const hideBar = useCallback(() => {
    setShowing(false);
  }, []);

  const onAction = useCallback(
    async (item: unknown, query: (inputEl: string) => void) => {
      const input: HTMLInputElement | null = omnibarInput();

      if (!input) return;
      if (isTreeNode(item)) {
        try {
          await Tree.onAction(input, item, query);
          if (item.closeBar) {
            hideBar();
          }
        } catch (e) {
          handleError(e);
        }
      }
    },
    [hideBar],
  );

  useEffect(() => {
    const input: HTMLInputElement | null = omnibarInput();
    if (showing) {
      if (input) {
        input.focus();
        input.select();
      }
    }
  }, [showing]);

  return (
    <div className={css.base} style={{ display: showing ? 'unset' : 'none' }}>
      <div className={css.backdrop} onClick={hideBar} />
      <div className={css.bar} id="omnibar">
        <OmnibarNpm<BaseNode>
          autoFocus={true}
          extensions={[Tree.extension]}
          maxResults={7}
          placeholder='Type a command or "help" for more info.'
          render={TreeNode}
          onAction={onAction}
        />
      </div>
    </div>
  );
};

export default Omnibar;
