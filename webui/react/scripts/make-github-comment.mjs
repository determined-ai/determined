const message = process.argv[2];
if (!message) {
  throw new Error('node make-github-comment.mjs <message>');
}

const projectUsername = process.env['CIRCLE_PROJECT_USERNAME'];
const reponame = process.env['CIRCLE_PROJECT_REPONAME'];
const branch = process.env['CIRCLE_BRANCH'];
const username = process.env['GITHUB_USERNAME'];
const password = process.env['GITHUB_TOKEN'];

const headers = {
  Accept: 'application/json',
  Authorization: `Bearer ${password}`,
};

// get attached pr:
const prUrl = new URL(`https://api.github.com/repos/${projectUsername}/${reponame}/pulls`);
prUrl.searchParams.set('state', 'open');
prUrl.searchParams.set('head', `${projectUsername}:${branch}`);
const [prPayload] = await fetch(prUrl.toString(), { headers }).then((r) => r.json());
if (!prPayload) {
  console.error('No PR found, not reporting artifact to github');
  process.exit();
}

const commentsUrl = new URL(prPayload.comments_url);
const commentsPayload = await fetch(commentsUrl.toString(), { headers }).then((r) => r.json());
// TODO: paginate in case we're unlucky and the comment to update isn't in the first page
const [commentToUpdate] = commentsPayload.filter((comment) => comment.user.login === username);
const commentOptions = {
  body: JSON.stringify({ body: message }),
  headers: {
    ...headers,
    'Content-Type': 'application/json',
  },
};
if (commentToUpdate) {
  commentOptions.method = 'PATCH';
  const updateUrl = new URL(commentToUpdate.url);
  fetch(updateUrl.toString(), commentOptions).then((r) => r.json());
} else {
  commentOptions.method = 'POST';
  await fetch(commentsUrl.toString(), commentOptions).then((r) => r.json());
}
