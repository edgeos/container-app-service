node {

    stage "SCM Checkout"
    checkout scm

    stage "Clean Existing Images"
    sh '''
        echo "TODO"
    '''

    stage "Pre-Build Steps"
    sh '''
        (cd ci-scripts && git pull) || git clone https://$(cat /home/jenkins/git_token)@github.build.ge.com/PredixEdge/ci-scripts.git
        source ci-scripts/prebuild.sh
    '''

    stage "Build"
    sh '''
        echo "TODO"
    '''

    stage "Image"
    sh '''
        echo "TODO"
    '''

    stage "Test"
    sh '''
        echo "TODO"
    '''

    stage "Scan"
    sh '''
        echo "TODO"
    '''

    stage "Post-Build Steps"
    sh '''
        source ci-scripts/postbuild.sh
    '''

    stage "Clean Existing Images"
    sh '''
        echo "TODO"
    '''

    stage "Final Step"
    sh '''
        source ci-scripts/finally.sh
    '''

}
