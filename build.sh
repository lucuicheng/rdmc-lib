#!/bin/sh

cd .
PROJECT_PATH=$(pwd)

build() {
  FLAG=$1
  OS=$2
  ARCH=$3
  NAME=$4
  VERSION=$5

  # build env
  echo "-----------$PROJECT_PATH build to $OS $ARCH-------------"
  GOOS=$OS GOARCH=$ARCH go build -ldflags "-X rdmc/pkg.AppVersion=$VERSION" -o ./dist/$FLAG/$NAME $NAME.go
  cd ./dist/$FLAG/
#  upx $NAME
  echo "$FLAG build done!"

  # 判断文件名是否包含 "-"
  if [[ $NAME == *-* ]]; then
    # 如果包含 "-"，截取后半部分
    new_filename=$(echo "$NAME" | cut -d'-' -f2-)
    mv $NAME $new_filename
    echo "new bin name is : $new_filename"
  else
    # 如果不包含 "-"，不处理
    echo "bin don't contains \"-\", no need rename"
  fi
}

local_build() {
  FLAG=$1
  OS=$2
  ARCH=$3
  NAME=$4
  VERSION=$5

  # build env
  #  echo "-----------$PROJECT_PATH build to $OS $ARCH-------------"
  echo $(pwd)
  go build -ldflags "-X rdmc/pkg.AppVersion=$VERSION" -o $PROJECT_PATH/dist/$FLAG/$NAME $NAME.go
  cd $PROJECT_PATH/dist/$FLAG/
  #  upx $NAME$VERSION
  echo "$FLAG build done!"

  # 判断文件名是否包含 "-"
  if [[ $NAME == *-* ]]; then
    # 如果包含 "-"，截取后半部分
    new_filename=$(echo "$NAME" | cut -d'-' -f2-)
    mv $NAME $new_filename
    echo "new bin name is : $new_filename"
  else
    # 如果不包含 "-"，不处理
    echo "bin don't contains \"-\", no need rename"
  fi

  # 本机安装
  if [[ $FLAG == macos ]]; then
    chmod a+x $PROJECT_PATH/dist/$FLAG/$EXPORT_NAME
    cp $PROJECT_PATH/dist/$FLAG/$EXPORT_NAME /usr/local/bin/$EXPORT_NAME
    cp $PROJECT_PATH/dist/$FLAG/$EXPORT_NAME /opt/rdmc/lib/$EXPORT_NAME
    echo "$FLAG install done!"
  fi

  cd $PROJECT_PATH # 回到项目目录
}

batch() {
  #NAME=rsdefender
  #-1.0.2
  EXPORT_NAME=$1
  VERSION=$2

  # 本机 编译，默认为 macos 环境
  local_build macos darwin amd64 main-$EXPORT_NAME $VERSION
  echo ""

  cd $PROJECT_PATH
  build linux linux amd64 main-$EXPORT_NAME $VERSION
#  cp $PROJECT_PATH/dist/linux/$EXPORT_NAME /media/psf/Share/lib/$EXPORT_NAME

  echo ""
  cd $PROJECT_PATH
  build windows windows amd64 main-$EXPORT_NAME $VERSION

}

nowtime=$(date '+%Y/%m/%d@%H:%M:%S')
echo $nowtime


#batch oracle 1.0.3
#batch mysql 1.0.2
#batch baits 1.0.5
#batch md5 1.0.2
#batch scan 1.0.2
batch extract '1.0.5_$nowtime'
#batch frs '1.0.4_$nowtime'
#batch license '1.0.0_$nowtime'
#batch mssql '1.0.0'
#batch netdisk '1.0.0'
