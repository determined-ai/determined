import React from 'react';

import css from './InfoBox.module.scss';

interface Row {
  label: string;
  info?: React.ReactNode;
}

interface Props {
  rows: Row[];
}

export const renderRow = ({ label, info }: Row): React.ReactNode => {
  if (info === undefined) return <></>;
  return (
    <tr key={label}>
      <td className={css.label}>{label}</td>
      <td>
        {[ 'string', 'number' ].includes(typeof info) ?
          <span>{info}</span> : info
        }
      </td>
    </tr>
  );
};

const InfoBox: React.FC<Props> = ({ rows }: Props) => {

  return (
    <table className={css.base}>
      <tbody>
        {rows.map(renderRow)}
      </tbody>
    </table>
  );
};

export default InfoBox;
