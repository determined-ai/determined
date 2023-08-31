import React from 'react';

import css from './Loader.module.scss';

const Loader: React.FC = () => {
  return (
    <div className={css.base}>
      <div className={css.loader}>
        <div className={css.row}>
          <span />
          <span />
          <span />
        </div>
        <div className={css.row}>
          <span />
          <span />
          <span />
        </div>
        <div className={css.row}>
          <span />
          <span />
          <span />
        </div>
      </div>
    </div>
  );
};

export default Loader;
