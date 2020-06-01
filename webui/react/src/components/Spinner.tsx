import React from 'react';

import Icon from 'components/Icon';

import css from './Spinner.module.scss';

interface Props {
  fullPage?: boolean;
  opaque?: boolean;
}

const Spinner: React.FC<Props> = (props: Props) => {
  const classes = [ css.base ];

  if (props.fullPage) classes.push(css.fullPage);
  if (props.opaque) classes.push(css.opaque);

  return (
    <div className={classes.join(' ')}>
      <div className={css.spin}>
        <Icon name="spinner" size="large" />
      </div>
    </div>
  );
};

export default Spinner;
