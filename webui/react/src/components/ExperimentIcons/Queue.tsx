import React from 'react';

import css from './Queue.module.scss';

const Queue: React.FC = (() => {
    return (
      <div className={css.base}>
        <div className={css.spinner}>
          <div className={css.inner_spinner} />
        </div>
      </div>);
});

export default Queue;
