import nodeFetch from 'node-fetch';

const message = process.argv[2];
if (!message) {
  throw new Error('node make-github-comment.mjs <message>');
}
const projectUsername = process.env['CIRCLE_PROJECT_USERNAME'];
const reponame = process.env['CIRCLE_PROJECT_REPONAME'];
const branch = process.env['CIRCLE_BRANCH'];
const username = process.env['GITHUB_USERNAME'];
const password = process.env['GITHUB_TOKEN'];
const jobId = process.env['CIRCLE_WORKFLOW_JOB_ID'];
// TODO: get rid of ashton fallback
const login = process.env['GITHUB_LOGIN'] || 'ashtonG';

console.error(projectUsername);
console.error(reponame);
console.error(branch);
console.error(username);
console.error(password);
console.error(jobId);
console.error(login);

// get attached pr:
const prUrl = new URL(`https://api.github.com/repos/${projectUsername}/${reponame}/pulls`);
prUrl.searchParams.set('state', 'open');
prUrl.searchParams.set('head', `${projectUsername}:${branch}`);
prUrl.username = username;
prUrl.password = password;
const [prPayload] = await nodeFetch(prUrl.toString()).then((r) => r.json());
if (!prPayload) {
  console.error('No PR found, not reporting artifact to github');
}

const commentsUrl = new URL(prPayload.comments_url);
commentsUrl.username = username;
commentsUrl.password = password;
const commentsPayload = await nodeFetch(commentsUrl.toString()).then((r) => r.json());
// TODO: paginate in case we're unlucky and the comment to update isn't in the first page
const [commentToUpdate] = commentsPayload.filter((comment) => comment.user.login === login);
const artifactUrl = `https://output.circle-artifacts.com/output/job/${jobId}/artifacts/0/webui/react/screenshot-summary.html`;
const comment = `Hello! DesignKit diffs are available for you to view [here](${artifactUrl})`;
const commentOptions = {
  body: JSON.stringify({ body: comment }),
  headers: {
    'Accept': 'application/json',
    'Content-Type': 'application/json',
  },
};
if (commentToUpdate) {
  commentOptions.method = 'patch';
  const updateUrl = new URL(commentToUpdate.url);
  updateUrl.username = username;
  updateUrl.password = password;
  const updatePayload = await nodeFetch(updateUrl.toString(), commentOptions).then((r) => r.json());
  console.error(updatePayload);
} else {
  commentOptions.method = 'post';
  const createPayload = await nodeFetch(commentsUrl.toString(), commentOptions).then((r) =>
    r.json(),
  );
  console.error(createPayload);
}
