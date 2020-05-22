import React from 'react';

import Icon from 'components/Icon';

import css from './Spinner.module.scss';

interface Props {
  fillContainer?: boolean;
  fullPage?: boolean;
  opaque?: boolean;
  shade?: boolean;
}

const Spinner: React.FC<Props> = (props: Props) => {
  const classes = [ css.base ];

  if (props.fillContainer) classes.push(css.fillContainer);
  if (props.fullPage) classes.push(css.fullPage);
  if (props.opaque) classes.push(css.opaque);
  if (props.shade) classes.push(css.shade);

  return (
    <div className={classes.join(' ')}>
      <div className={css.spin}>
        <Icon name="spinner" size="large" />
      </div>
    </div>
  );
};

export default Spinner;
