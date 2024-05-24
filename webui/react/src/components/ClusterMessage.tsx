import React from 'react';

import { ClusterMessage } from 'stores/determinedInfo';

import css from './ClusterMessage.module.scss';

interface Props {
  message?: ClusterMessage;
}

const ClusterMessageBanner: React.FC<Props> = ({ message }) => {
  return message ? (
    <>
      <div className={css.base}>
        <span>
          <span className={css.clusterMessageLabel} data-testid="admin-msg">
            Message from Admin:
          </span>{' '}
          <span data-testid="cluster-msg">{message.message}</span>
        </span>
      </div>
    </>
  ) : null;
};

export default ClusterMessageBanner;
