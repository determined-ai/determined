import React from 'react';

import css from './Spinner.module.scss';

interface Props {
    type: 'bowtie' | 'half'
}

const Spinner: React.FC<Props> = (({ type }) => {
    const classnames = [css.spinner, css[`spinner__${type}`]];
    return <div className={css.base}><div className={classnames.join(' ')} /></div>;
});

export default Spinner;
