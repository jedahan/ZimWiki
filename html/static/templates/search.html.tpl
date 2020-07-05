{{  define "content" }}

    <center>
    <div>
        <h1> Search results for "{{ .QueryText }}" </h1>
        <span style="width:100%;text-align:center">
            [in {{ .SearchSource }}]
        </span>
    </div>
    </center>

    <br><br>

    {{ range .Results }}
        <div class="row" style="padding-bottom: 15px;">
                <div class="col">
                    <div class="card">
                        <div class="card-body">
                            <h4 class="card-title">{{ .Title }}</h4>
                            <p class="card-text">Here a text</p><a class="card-link float-right" href="{{ .Link }}">Open</a></div>
                    </div>
                </div>
            </div>
    {{ end }}

{{ end }}