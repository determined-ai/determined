import OmnibarNpm from 'omnibar';
import React, { useEffect } from 'react';
import { GlobalHotKeys } from 'react-hotkeys';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { exposeStore } from 'omnibar/exposedStore';
import BaseRenderer from 'omnibar/modifiers/anchor/BaseRenderer';
import * as Tree from 'omnibar/Tree';

import { BaseNode } from './AsyncTree';
import css from './Component.module.scss';

const globalKeymap = { HIDE_OMNIBAR: [ 'esc' ] };

const Omnibar: React.FC = () => {
  const storeDispatch = useStoreDispatch();
  const { omnibar } = useStore();
  const globalKeyHandler = {
    HIDE_OMNIBAR: (): void => {
      storeDispatch({ type: StoreAction.HideOmnibar });
    },
    SHOW_OMNIBAR: (): void => storeDispatch({ type: StoreAction.ShowOmnibar }),
  };

  useEffect(() => {
    exposeStore({ dispatch: storeDispatch, state: { omnibar } });
  }, [ storeDispatch, omnibar ]);

  return (
    <div className={css.base}>
      <div className={css.bar} id="omnibar">
        <OmnibarNpm<BaseNode>
          autoFocus={true}
          extensions={[ Tree.extension ]}
          maxResults={7}
          placeholder='Type a command or "help" for more info.'
          render={BaseRenderer}
          rootStyle={{ width: 'calc(max(40vw, 30rem))' }}
          onAction={Tree.onAction as any} />
      </div>
      <GlobalHotKeys handlers={globalKeyHandler} keyMap={globalKeymap} />
    </div>
  );
};

export const keymap = { HIDE_OMNIBAR: 'esc' };

export default Omnibar;
