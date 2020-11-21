import React, { useEffect, useRef } from 'react';

import { InfoDecorator } from 'storybook/ContextDecorators';
import { downloadText, simulateLogsDownload } from 'utils/browser';

import LogViewer, { LogViewerHandles } from './LogViewer';

export default {
  component: LogViewer,
  decorators: [ InfoDecorator ],
  parameters: { layout: 'fullscreen' },
  title: 'LogViewer',
};

const messageWithTags = `continuing trial: <COMPUTE_VALIDATION_METRICS: (10,10,30)>
    experiment-id="10" id="c075adcd-5c31-4726-a5b3-0b87e141f1fe"
    system="master" trial-id="10" type="trial"
`;

export const Default = (): React.ReactNode => {
  const logsRef = useRef<LogViewerHandles>(null);

  useEffect(() => {
    /* eslint-disable max-len */
    if (logsRef.current) logsRef.current?.addLogs([
      { id: 0, message: 'Simple one liner.', time: '2020-06-02T21:48:07.456381-06:00' },
      { id: 1, message: 'Another line', time: '2020-06-02T21:48:08.456381-06:00' },
      { id: 2, message: 'Example of a really long line. Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry\'s standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.', time: '2020-06-02T21:48:09.456381-06:00' },
      { id: 3, message: 'Example of multi-line log with newlines\nanother line\nanother line\nanother line', time: '2020-06-02T21:48:10.456381-06:00' },
      { id: 4, message: messageWithTags, time: '2020-06-02T21:48:12.456389-06:00' },
    ]);
  });

  return <LogViewer pageProps={{ title: 'Default' }} ref={logsRef} />;
};

const ansiText = `Standard colors:
[30m  30[0m[31m  31[0m[32m  32[0m[33m  33[0m[34m  34[0m[35m  35[0m[36m  36[0m
Intense colors:
[90m  90[0m[91m  91[0m[92m  92[0m[93m  93[0m[94m  94[0m[95m  95[0m[96m  96[0m
256 colors:
[38;5;1m   1[0m[38;5;2m   2[0m[38;5;3m   3[0m[38;5;4m   4[0m[38;5;5m   5[0m[38;5;6m   6[0m[38;5;7m   7[0m[38;5;8m   8[0m
[38;5;9m   9[0m[38;5;10m  10[0m[38;5;11m  11[0m[38;5;12m  12[0m[38;5;13m  13[0m[38;5;14m  14[0m[38;5;15m  15[0m[38;5;16m  16[0m
[38;5;16m  16[0m[38;5;17m  17[0m[38;5;18m  18[0m[38;5;19m  19[0m[38;5;20m  20[0m[38;5;21m  21[0m
[38;5;22m  22[0m[38;5;23m  23[0m[38;5;24m  24[0m[38;5;25m  25[0m[38;5;26m  26[0m[38;5;27m  27[0m
[38;5;28m  28[0m[38;5;29m  29[0m[38;5;30m  30[0m[38;5;31m  31[0m[38;5;32m  32[0m[38;5;33m  33[0m
[38;5;34m  34[0m[38;5;35m  35[0m[38;5;36m  36[0m[38;5;37m  37[0m[38;5;38m  38[0m[38;5;39m  39[0m
[38;5;40m  40[0m[38;5;41m  41[0m[38;5;42m  42[0m[38;5;43m  43[0m[38;5;44m  44[0m[38;5;45m  45[0m
[38;5;46m  46[0m[38;5;47m  47[0m[38;5;48m  48[0m[38;5;49m  49[0m[38;5;50m  50[0m[38;5;51m  51[0m
[38;5;52m  52[0m[38;5;53m  53[0m[38;5;54m  54[0m[38;5;55m  55[0m[38;5;56m  56[0m[38;5;57m  57[0m
[38;5;58m  58[0m[38;5;59m  59[0m[38;5;60m  60[0m[38;5;61m  61[0m[38;5;62m  62[0m[38;5;63m  63[0m
[38;5;64m  64[0m[38;5;65m  65[0m[38;5;66m  66[0m[38;5;67m  67[0m[38;5;68m  68[0m[38;5;69m  69[0m
[38;5;70m  70[0m[38;5;71m  71[0m[38;5;72m  72[0m[38;5;73m  73[0m[38;5;74m  74[0m[38;5;75m  75[0m
[38;5;76m  76[0m[38;5;77m  77[0m[38;5;78m  78[0m[38;5;79m  79[0m[38;5;80m  80[0m[38;5;81m  81[0m
[38;5;82m  82[0m[38;5;83m  83[0m[38;5;84m  84[0m[38;5;85m  85[0m[38;5;86m  86[0m[38;5;87m  87[0m
[38;5;88m  88[0m[38;5;89m  89[0m[38;5;90m  90[0m[38;5;91m  91[0m[38;5;92m  92[0m[38;5;93m  93[0m
[38;5;94m  94[0m[38;5;95m  95[0m[38;5;96m  96[0m[38;5;97m  97[0m[38;5;98m  98[0m[38;5;99m  99[0m
[38;5;100m 100[0m[38;5;101m 101[0m[38;5;102m 102[0m[38;5;103m 103[0m[38;5;104m 104[0m[38;5;105m 105[0m
[38;5;106m 106[0m[38;5;107m 107[0m[38;5;108m 108[0m[38;5;109m 109[0m[38;5;110m 110[0m[38;5;111m 111[0m
[38;5;112m 112[0m[38;5;113m 113[0m[38;5;114m 114[0m[38;5;115m 115[0m[38;5;116m 116[0m[38;5;117m 117[0m
[38;5;118m 118[0m[38;5;119m 119[0m[38;5;120m 120[0m[38;5;121m 121[0m[38;5;122m 122[0m[38;5;123m 123[0m
[38;5;124m 124[0m[38;5;125m 125[0m[38;5;126m 126[0m[38;5;127m 127[0m[38;5;128m 128[0m[38;5;129m 129[0m
[38;5;130m 130[0m[38;5;131m 131[0m[38;5;132m 132[0m[38;5;133m 133[0m[38;5;134m 134[0m[38;5;135m 135[0m
[38;5;136m 136[0m[38;5;137m 137[0m[38;5;138m 138[0m[38;5;139m 139[0m[38;5;140m 140[0m[38;5;141m 141[0m
[38;5;142m 142[0m[38;5;143m 143[0m[38;5;144m 144[0m[38;5;145m 145[0m[38;5;146m 146[0m[38;5;147m 147[0m
[38;5;148m 148[0m[38;5;149m 149[0m[38;5;150m 150[0m[38;5;151m 151[0m[38;5;152m 152[0m[38;5;153m 153[0m
[38;5;154m 154[0m[38;5;155m 155[0m[38;5;156m 156[0m[38;5;157m 157[0m[38;5;158m 158[0m[38;5;159m 159[0m
[38;5;160m 160[0m[38;5;161m 161[0m[38;5;162m 162[0m[38;5;163m 163[0m[38;5;164m 164[0m[38;5;165m 165[0m
[38;5;166m 166[0m[38;5;167m 167[0m[38;5;168m 168[0m[38;5;169m 169[0m[38;5;170m 170[0m[38;5;171m 171[0m
[38;5;172m 172[0m[38;5;173m 173[0m[38;5;174m 174[0m[38;5;175m 175[0m[38;5;176m 176[0m[38;5;177m 177[0m
[38;5;178m 178[0m[38;5;179m 179[0m[38;5;180m 180[0m[38;5;181m 181[0m[38;5;182m 182[0m[38;5;183m 183[0m
[38;5;184m 184[0m[38;5;185m 185[0m[38;5;186m 186[0m[38;5;187m 187[0m[38;5;188m 188[0m[38;5;189m 189[0m
[38;5;190m 190[0m[38;5;191m 191[0m[38;5;192m 192[0m[38;5;193m 193[0m[38;5;194m 194[0m[38;5;195m 195[0m
[38;5;196m 196[0m[38;5;197m 197[0m[38;5;198m 198[0m[38;5;199m 199[0m[38;5;200m 200[0m[38;5;201m 201[0m
[38;5;202m 202[0m[38;5;203m 203[0m[38;5;204m 204[0m[38;5;205m 205[0m[38;5;206m 206[0m[38;5;207m 207[0m
[38;5;208m 208[0m[38;5;209m 209[0m[38;5;210m 210[0m[38;5;211m 211[0m[38;5;212m 212[0m[38;5;213m 213[0m
[38;5;214m 214[0m[38;5;215m 215[0m[38;5;216m 216[0m[38;5;217m 217[0m[38;5;218m 218[0m[38;5;219m 219[0m
[38;5;220m 220[0m[38;5;221m 221[0m[38;5;222m 222[0m[38;5;223m 223[0m[38;5;224m 224[0m[38;5;225m 225[0m
[38;5;226m 226[0m[38;5;227m 227[0m[38;5;228m 228[0m[38;5;229m 229[0m[38;5;230m 230[0m[38;5;231m 231[0m
[38;5;232m 232[0m[38;5;233m 233[0m[38;5;234m 234[0m[38;5;235m 235[0m[38;5;236m 236[0m[38;5;237m 237[0m
[38;5;238m 238[0m[38;5;239m 239[0m[38;5;240m 240[0m[38;5;241m 241[0m[38;5;242m 242[0m[38;5;243m 243[0m
[38;5;244m 244[0m[38;5;245m 245[0m[38;5;246m 246[0m[38;5;247m 247[0m[38;5;248m 248[0m[38;5;249m 249[0m
[38;5;250m 250[0m[38;5;251m 251[0m[38;5;252m 252[0m[38;5;253m 253[0m[38;5;254m 254[0m[38;5;255m 255[0m
`;

export const Ansi = (): React.ReactNode => {
  const logsRef = useRef<LogViewerHandles>(null);

  useEffect(() => {
    /* eslint-disable max-len */
    if (logsRef.current) logsRef.current?.addLogs([
      { id: 0, message: 'example of logs with ANSI color codes', time: '2020-06-02T21:48:07.456381-06:00' },
      { id: 1, message: ansiText, time: '2020-06-02T21:48:08.456381-06:00' },
    ]);
  });

  return <LogViewer pageProps={{ title: 'ANSI Characters' }} ref={logsRef} />;
};

export const DefaultDownload = (): React.ReactNode => {
  return <button onClick={() => downloadText('default-logs.txt', [ messageWithTags ])}>
    Download Default Logs
  </button>;
};

export const AnsiDownload = (): React.ReactNode => {
  return <button onClick={() => downloadText('ansi-logs.txt', [ ansiText ])}>
    Download Ansi Logs
  </button>;
};

export const SimulatedDownload = (): React.ReactNode => {
  const sizeRef = useRef<HTMLInputElement>(null);
  return <div>
    <div>
      <label>Number of log characters to generate and download (rounded up): </label>
      <input placeholder="Log size in characters" ref={sizeRef} type="number" />
    </div>
    <button onClick={() => simulateLogsDownload(parseInt(sizeRef.current?.value || '100000'))}>
      Download
    </button>
  </div>;
};
