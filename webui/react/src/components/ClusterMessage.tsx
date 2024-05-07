import React from 'react';

import { ClusterMessage } from 'stores/determinedInfo';

import css from './ClusterMessage.module.scss';

interface Props {
  message?: ClusterMessage;
}

const ClusterMessage: React.FC<Props> = ({ message }) => {
  return message ? (
    <>
      <div className={css.base}>
        <span>
          <span className={css.clusterMessageLabel}>Message from Admin:</span>{' '}
          <span>{message.message}</span>
        </span>
      </div>
    </>
  ) : null;
};

export default ClusterMessage;
