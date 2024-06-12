import React from 'react';

import { ClusterMessage } from 'stores/determinedInfo';

import css from './ClusterMessage.module.scss';

interface Props {
  message?: ClusterMessage;
}

const ClusterMessageBanner: React.FC<Props> = ({ message }) => {
  return message && message.message.length > 0 ? (
    <div className={css.base}>
      <span className={css.clusterMessageLabel} data-testid="admin-msg">
        Message from Admin:
      </span>
      <span data-testid="cluster-msg">{message.message}</span>
    </div>
  ) : null;
};

export default ClusterMessageBanner;
