package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

func checkErr(e error) {
	// Standard error check
	if e != nil {
		log.Println(e)
	}
}

func checkExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func getWd() (path string) {
	// Get working directory
	path, err := os.Getwd()
	path = path + "\\"
	checkErr(err)
	return
}

func getArgs() (file string, help bool, clean bool, nosend bool) {
	// Parse input arguments
	f := flag.String("f", "", "Input file")
	h := flag.Bool("h", false, "Get help")
	c := flag.Bool("c", false, "Clean files")
	n := flag.Bool("n", false, "Don't send to Google Speech-to-Text")

	flag.Parse()
	help = *h
	clean = *c
	file = *f
	nosend = *n
	return
}

func cleanUp(file string) {
	// Remove leftover files
	outfile := strings.TrimSuffix(file, filepath.Ext(file)) + ".wav"

	if checkExists(outfile) {
		os.Remove(outfile)
		fmt.Printf("Removed %v\n", outfile)
	}
	if checkExists(file + ".bak") {
		os.Remove(outfile)
		fmt.Printf("Removed %v\n", file+".bak")
	}
}

func copy(src string, dst string) {
	// Read all content of src to data
	data, err := ioutil.ReadFile(src)
	checkErr(err)
	// Write data to dst
	err = ioutil.WriteFile(dst, data, 0644)
	checkErr(err)
}

func fmmpegToWav(file string) (outfile string, err error) {
	// Utilise ffmpeg to convert video files to .wav audio for transcription
	fmt.Printf("\nBacking up %v to %v\n\n", file, file+".bak")
	copy(file, file+".bak")

	fmt.Println("Converting file to .wav audio...")
	outfile = strings.TrimSuffix(file, filepath.Ext(file)) + ".wav"
	cmd := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "warning", "-i", file, "-ac", "1", "-ar", "16000", "-acodec", "pcm_s16le", outfile)
	cmdOut, err := cmd.CombinedOutput()
	checkErr(err)

	fmt.Println(string(cmdOut))
	fmt.Printf("Successfully converted to %v. See warnings above (if any).\n", outfile)
	return
}

func wav2letter(file string) {
	// Definite WIP. Needs error checking and all sorts. Had it outputting the stdout as well as piping but couldn't snag to find when the process finished.
	str := string(`run --rm -v ` + getWd() + `:/root/host/ --ipc=host --name w2l swilcock0/wav2letter_ex_go sh -c`)
	str2 := string(` cat /root/host/` + file + ` | /root/wav2letter/build/inference/inference/examples/simple_streaming_asr_example --input_files_base_path /root/host/model > /root/host/+` + file + `_w2ltranscription.txt`)

	strs := append(strings.Split(str, " "), str2)
	//fmt.Println("Args : " + strings.Join(strs, ","))

	cmd := exec.Command("docker", strs...)
	checkErr(cmd.Run())

	fmt.Println("Done! Hopefully... check files")

}

func sendGCS(f *os.File, client *speech.Client, gcsURI string) error {
	ctx := context.Background()

	// Send the contents of the audio file with the encoding and
	// and sample rate information to be transcripted.
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    "en-US",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: gcsURI},
		},
	}

	op, err := client.LongRunningRecognize(ctx, req)
	if err != nil {
		return err
	}
	resp, err := op.Wait(ctx)
	if err != nil {
		return err
	}

	// Print the results.
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			f.WriteString(fmt.Sprintf("\"%v\" (confidence=%3f)\n", alt.Transcript, alt.Confidence))
		}
	}
	return nil
}

func main() {
	// Get working directory
	path := getWd()

	// Get filename of video from argument
	file, help, clean, nosend := getArgs()

	if clean == true {
		cleanUp(file)
	}

	// checkErr filename given
	if (file == "") || (help == true) {
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("File: " + path + file)

	// Convert file to audio
	audioFile, err := fmmpegToWav(file)
	checkErr(err)

	if nosend != false {
		os.Exit(0)
	}
	ctx := context.Background()
	c, err := speech.NewClient(ctx)
	checkErr(err)
	defer c.Close()

	f, err := os.Create(audioFile + "_transcription.txt")
	defer f.Close()

	wav2letter(audioFile)
	// TODO: Implement gcs
	//err = sendGCS(f, c, audioFile, gcsURI)
	//checkErr(err)

}
