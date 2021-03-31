import OmnibarNpm from 'omnibar';
import React from 'react';

import * as Tree from 'omnibar/Tree';

import css from './Component.module.scss';

interface ItemProps<T> {
  item: T;
  isSelected: boolean;
  isHighlighted: boolean;
}

const ItemRenderer = (p: ItemProps<any>) => {
  return (
    <div>{p.item.name}</div>
  );
};

const Omnibar: React.FC = () => {
  return (
    <div className={css.base}>
      <div className={css.bar} id="omnibar">
        {/* <OmnibarNpm
        autoFocus={true}
        extensions={[ funcExt ]}
        placeholder="Type a function name"
        onAction={funcOnAction}
        /> */}
        <OmnibarNpm
          autoFocus={true}
          extensions={[ Tree.extension ]}
          maxResults={7}
          placeholder="Type away.."
          onAction={Tree.onAction}
        /*render={ItemRenderer}*/ />
      </div>
    </div>
  );
};

export const keymap = { HIDE_OMNIBAR: 'esc' };

export default Omnibar;
