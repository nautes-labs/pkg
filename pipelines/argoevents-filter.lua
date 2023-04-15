pipelineDirName = "pipelines/"
deploymentDirName = "deployments/"
function startsWith(str, prefix)
  return string.sub(str, 1, string.len(prefix)) == prefix
end
function isSkip(files)
  for a = 1, #files do
    file = files[a]
    if startsWith(file, pipelineDirName) or startsWith(file, deploymentDirName) then
      return true
    end
  end
  return false
end
for i = 1, #event.body.commits do
  commit = event.body.commits[i]
  if isSkip(commit.added) or isSkip(commit.modified) or isSkip(commit.removed) then
    return false
  end
end
return true