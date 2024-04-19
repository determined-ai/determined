import React, { useEffect } from 'react';

import { ClusterMessage } from 'stores/determinedInfo';

import css from './ClusterMessage.module.scss';

interface Props {
  message?: ClusterMessage;
}

const ClusterMessage: React.FC<Props> = ({ message }) => {
  useEffect(() => {
    console.log('message component', message);
  }, [message]);

  const msg = message
    ? message.message +
      'adding a bunch of text to see what it will be like with a longer cluster message. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones. Adding even more to test extremely long ones.'
    : '';
  const trimmedMsg = msg.substring(0, 100);

  return message ? (
    <>
      {/* <div className={css.placeHolder}>&nbsp;</div> */}
      <div className={css.base}>
        <span className={css.clusterMessageLabel}>Message from Admin</span>:{trimmedMsg}
      </div>
    </>
  ) : null;
};

export default ClusterMessage;
