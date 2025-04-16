# Check if required variables are set
if [ -z "${SIGNING_CERTIFICATE_NAME}" ]; then
  echo "Error: SIGNING_CERTIFICATE_NAME is not set"
  exit 1
fi


codesign --force --sign "$SIGNING_CERTIFICATE_NAME" --options runtime --timestamp pkg/swift-argo/ArgoTrading.xcframework