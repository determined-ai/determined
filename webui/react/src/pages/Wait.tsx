import queryString from 'query-string';
import React, { useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import Message from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import history from 'routes/history';
import { serverAddress } from 'routes/utils';
import { waitCommandReady } from 'wait';

interface Params {
  taskId: string;
  taskType: string;
}

interface Queries {
  eventUrl?: string;
  serviceAddr?: string;
}

const Wait: React.FC = () => {
  const { taskId, taskType } = useParams<Params>();
  const [ message, setMessage ] = useState<string>();
  const { eventUrl, serviceAddr }: Queries = queryString.parse(location.search);

  useEffect(() => {
    if (!eventUrl || !serviceAddr) return;
    console.log(serviceAddr);
    waitCommandReady(eventUrl)
      .then(() => {
        window.location.assign(serverAddress(serviceAddr));
      })
      .catch(() => {
        setMessage('Error'); // TODO
      });
  }, [ eventUrl, serviceAddr, setMessage ]);

  if (!eventUrl || !serviceAddr) return <Message title="missing required parameters" />;

  return (
    <Page id="wait" title={`Waiting for ${taskType} ${taskId}`}>
      <Message title={message || 'Loading'} />

      <Spinner />
    </Page>
  );
};

export default Wait;
